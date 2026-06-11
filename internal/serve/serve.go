package serve

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spekk-ai/spekk-cli/internal/cli"
)

const defaultPort = 3118

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if strings.HasPrefix(origin, "chrome-extension://") {
		return true
	}
	for _, prefix := range []string{"http://localhost", "http://127.0.0.1", "http://[::1]"} {
		if origin == prefix || strings.HasPrefix(origin, prefix+":") || strings.HasPrefix(origin, prefix+"/") {
			return true
		}
	}
	return false
}

// generateNonce creates a cryptographically random 32-byte hex string.
func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Options configures the serve command.
type Options struct {
	Port    int
	Host    string
	Verbose bool
}

// connection tracks a single WebSocket client and its Claude subprocess.
type connection struct {
	id     int
	ws     *websocket.Conn
	claude *exec.Cmd
	mu     sync.Mutex // protects ws writes
}

// sendJSON writes a JSON message to the WebSocket.
func (c *connection) sendJSON(msg outgoingMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ws.WriteJSON(msg)
}

// Run starts the WebSocket serve bridge.
func Run(opts Options, installDir string) error {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}
	if opts.Host == "" {
		opts.Host = "localhost"
	}

	debug := func(format string, args ...interface{}) {}
	if opts.Verbose {
		debug = func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, "[serve:debug] "+format+"\n", args...)
		}
	}

	// Generate session nonce
	nonce, err := generateNonce()
	if err != nil {
		return err
	}

	// Build the coach system prompt
	coachPrompt, err := buildServeCoachPrompt(installDir)
	if err != nil {
		return fmt.Errorf("building coach prompt: %w", err)
	}

	var (
		connMu      sync.Mutex
		connections = make(map[*websocket.Conn]*connection)
		connCounter int
	)

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// WebSocket endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Validate session nonce before upgrade
		if r.URL.Query().Get("nonce") != nonce {
			http.Error(w, "Forbidden: invalid or missing session nonce", http.StatusForbidden)
			return
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			debug("upgrade error: %v", err)
			return
		}

		connMu.Lock()
		connCounter++
		connID := connCounter
		connMu.Unlock()

		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "unknown"
		}
		fmt.Fprintf(os.Stderr, "[serve] Connection #%d opened (origin: %s)\n", connID, origin)

		conn := &connection{id: connID, ws: ws}
		_, connCancel := context.WithCancel(context.Background())
		defer connCancel()

		// Spawn Claude subprocess
		claude := exec.Command("claude",
			"-p",
			"--verbose",
			"--dangerously-skip-permissions",
			"--output-format", "stream-json",
			"--input-format", "stream-json",
			"--system-prompt", coachPrompt,
		)
		claude.Env = os.Environ()

		stdinPipe, err := claude.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[serve] #%d stdin pipe error: %v\n", connID, err)
			ws.Close()
			return
		}
		stdoutPipe, err := claude.StdoutPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[serve] #%d stdout pipe error: %v\n", connID, err)
			ws.Close()
			return
		}
		stderrPipe, err := claude.StderrPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[serve] #%d stderr pipe error: %v\n", connID, err)
			ws.Close()
			return
		}

		if err := claude.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "[serve] #%d failed to start claude: %v\n", connID, err)
			conn.sendJSON(outgoingMessage{
				Event: "coach:error",
				Data:  errorData{Message: "Failed to start agent: " + err.Error()},
			})
			ws.Close()
			return
		}

		conn.claude = claude
		connMu.Lock()
		connections[ws] = conn
		connMu.Unlock()

		var lastStatusKey string

		sendStatus := func(state, detail string) {
			key := state + ":" + detail
			if key == lastStatusKey {
				return
			}
			lastStatusKey = key
			debug("#%d sendStatus: %s %s", connID, state, detail)
			conn.sendJSON(outgoingMessage{
				Event: "coach:status",
				Data:  statusData{State: state, Detail: detail},
			})
		}

		// Stream Claude stdout → WebSocket
		go func() {
			defer connCancel()
			scanner := bufio.NewScanner(stdoutPipe)
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}

				var event map[string]interface{}
				if err := json.Unmarshal([]byte(line), &event); err != nil {
					debug("#%d stdout unmarshal error: %v", connID, err)
					continue
				}

				eventType, _ := event["type"].(string)
				debug("#%d claude event: %s", connID, eventType)

				switch eventType {
				case "assistant":
					sendStatus("idle", "")
					msg, _ := event["message"].(map[string]interface{})
					contentArr, _ := msg["content"].([]interface{})
					var textParts []string
					for _, c := range contentArr {
						cm, _ := c.(map[string]interface{})
						if cm["type"] == "text" {
							if t, ok := cm["text"].(string); ok {
								textParts = append(textParts, t)
							}
						}
					}
					if len(textParts) > 0 {
						sessionID, _ := event["session_id"].(string)
						debug("#%d -> assistant (%d chars)", connID, len(strings.Join(textParts, "")))
						conn.sendJSON(outgoingMessage{
							Event: "coach:assistant",
							Data:  assistantData{Content: strings.Join(textParts, ""), SessionID: sessionID},
						})
					}

				case "result":
					sendStatus("idle", "")
					sessionID, _ := event["session_id"].(string)
					resultStr, _ := event["result"].(string)
					isError, _ := event["is_error"].(bool)
					debug("#%d -> result (error=%v)", connID, isError)
					conn.sendJSON(outgoingMessage{
						Event: "coach:result",
						Data:  resultData{Content: resultStr, IsError: isError, SessionID: sessionID},
					})

				case "tool_use":
					tool, _ := event["tool"].(map[string]interface{})
					toolName, _ := tool["name"].(string)
					if toolName == "" {
						toolName, _ = event["name"].(string)
					}
					sendStatus("working", describeToolUse(toolName))

				case "tool_result":
					sendStatus("thinking", "")

				case "system", "init":
					sendStatus("thinking", "")
				}
			}
		}()

		// Forward stderr as error messages
		go func() {
			defer connCancel()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				msg := strings.TrimSpace(scanner.Text())
				if msg == "" {
					continue
				}
				fmt.Fprintf(os.Stderr, "[serve] #%d stderr: %.300s\n", connID, msg)
				conn.sendJSON(outgoingMessage{
					Event: "coach:error",
					Data:  errorData{Message: msg},
				})
			}
		}()

		// Wait for Claude to exit
		go func() {
			defer connCancel()
			err := claude.Wait()
			code := 1
			if err == nil {
				code = 0
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			}
			fmt.Fprintf(os.Stderr, "[serve] Claude process for connection #%d exited (code: %d)\n", connID, code)
			conn.sendJSON(outgoingMessage{
				Event: "coach:agent_exited",
				Data:  agentExitedData{Code: code},
			})
			ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Agent process exited"))
			ws.Close()

			connMu.Lock()
			delete(connections, ws)
			connMu.Unlock()
		}()

		// sendToClaude writes a formatted message to Claude's stdin.
		sendToClaude := func(formatted string) {
			if formatted == "" {
				return
			}
			debug("#%d <- formatted: %.300s", connID, formatted)
			sendStatus("thinking", "")
			stdinMsg := map[string]interface{}{
				"type":       "user",
				"message":    map[string]interface{}{"role": "user", "content": formatted},
				"session_id": "default",
			}
			data, err := json.Marshal(stdinMsg)
			if err != nil {
				debug("#%d marshal error: %v", connID, err)
				return
			}
			if _, err := stdinPipe.Write(append(data, '\n')); err != nil {
				debug("#%d stdin write error: %v", connID, err)
			}
		}

		// Read WebSocket messages
		for {
			_, msgBytes, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					fmt.Fprintf(os.Stderr, "[serve] Connection #%d closed\n", connID)
				} else {
					fmt.Fprintf(os.Stderr, "[serve] WebSocket error on connection #%d: %v\n", connID, err)
				}
				break
			}

			var incoming incomingMessage
			if err := json.Unmarshal(msgBytes, &incoming); err != nil {
				debug("#%d invalid message: %v", connID, err)
				continue
			}

			// Strip "coach:" prefix from event name
			op := strings.TrimPrefix(incoming.Event, "coach:")

			switch op {
			case "chat":
				debug("#%d <- coach:chat", connID)
				var data chatData
				if err := json.Unmarshal(incoming.Data, &data); err != nil {
					debug("#%d chat unmarshal error: %v", connID, err)
					continue
				}
				sendToClaude(formatChatMessage(data))

			case "element_selection":
				debug("#%d <- coach:elementSelection", connID)
				var data elementSelectionData
				if err := json.Unmarshal(incoming.Data, &data); err != nil {
					debug("#%d element_selection unmarshal error: %v", connID, err)
					continue
				}
				sendToClaude("[User selected an element]\n" + formatElementSelection(data))

			case "action_recording":
				debug("#%d <- coach:actionRecording", connID)
				var data actionRecordingData
				if err := json.Unmarshal(incoming.Data, &data); err != nil {
					debug("#%d action_recording unmarshal error: %v", connID, err)
					continue
				}
				sendToClaude("[User recorded browser actions]\n" + formatActionRecording(data))

			case "init":
				debug("#%d <- coach:init", connID)
				var data initData
				if err := json.Unmarshal(incoming.Data, &data); err != nil {
					debug("#%d init unmarshal error: %v", connID, err)
					continue
				}
				sendToClaude(formatInitMessage(data))
			}
		}

		// Clean up on disconnect
		connCancel()

		connMu.Lock()
		c, exists := connections[ws]
		if exists {
			delete(connections, ws)
		}
		connMu.Unlock()

		if exists && c.claude.Process != nil {
			c.claude.Process.Kill()
		}
	})

	// Find available port
	listener, err := listenOnPort(opts.Host, opts.Port, 10)
	if err != nil {
		return fmt.Errorf("binding to port: %w", err)
	}
	actualPort := listener.Addr().(*net.TCPAddr).Port

	server := &http.Server{Handler: mux}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	fmt.Fprintf(os.Stderr, "[serve] WebSocket server listening on ws://%s:%d\n", opts.Host, actualPort)
	fmt.Fprintf(os.Stderr, "[serve] Health check: http://%s:%d/health\n", opts.Host, actualPort)
	fmt.Fprintf(os.Stderr, "[serve] Connect URL: ws://%s:%d/?nonce=%s\n", opts.Host, actualPort, nonce)
	fmt.Fprintln(os.Stderr, "[serve] Press Ctrl+C to stop.")

	// Print nonce to stdout for programmatic consumption
	fmt.Printf(`{"nonce":"%s","port":%d}`+"\n", nonce, actualPort)

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\n[serve] Shutting down...")
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	// Kill all Claude processes
	connMu.Lock()
	for ws, conn := range connections {
		if conn.claude.Process != nil {
			conn.claude.Process.Kill()
		}
		ws.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))
		ws.Close()
	}
	connections = make(map[*websocket.Conn]*connection)
	connMu.Unlock()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(shutdownCtx)

	fmt.Fprintln(os.Stderr, "[serve] Server stopped.")
	return nil
}

// listenOnPort tries to listen on host:port, incrementing the port up to
// maxRetries times if in use.
func listenOnPort(host string, port, maxRetries int) (net.Listener, error) {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		addr := fmt.Sprintf("%s:%d", host, port+i)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// buildServeCoachPrompt builds the system prompt for serve sessions.
func buildServeCoachPrompt(installDir string) (string, error) {
	resolver := &cli.PromptResolver{
		HomeDir: homeDir(),
		Cwd:     cwdStr(),
	}

	message, err := resolver.CreateActivationMessage("coach")
	if err != nil {
		return "", err
	}

	// Append coordinator skill if available
	sr := &cli.SkillResolver{
		HomeDir:    homeDir(),
		Cwd:        cwdStr(),
		InstallDir: installDir,
	}
	skill := sr.ResolveSkill("coach", "coordinate")
	if skill != nil {
		message += "\n\n---\n\n**Available Skill: Coordinator**\n\n"
		message += "The coordinator skill is available for use when the user asks to plan work, organize branches, or analyze dependencies.\n"
		message += "Use it when you detect relevant triggers — do NOT execute it automatically on session start.\n"
		message += "\n<skill-reference>\n" + skill.Content + "\n</skill-reference>\n"
	}

	return message, nil
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func cwdStr() string {
	d, _ := os.Getwd()
	return d
}
