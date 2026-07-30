package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

func (w *Worker) invoke(ctx context.Context, cfg Config, conn *websocket.Conn, msg Message) {
	args := []string{
		"-p", "-",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
		"--dangerously-skip-permissions",
	}
	if msg.SessionID != "" {
		args = append(args, "--resume", msg.SessionID)
	}

	spoolDir, cleanupSpool, err := provisionSpool()
	if err != nil {
		sendError(ctx, conn, fmt.Sprintf("create conversation spool: %v", err))
		return
	}
	defer cleanupSpool()

	cmd := exec.Command("claude", args...)
	cmd.Dir = cfg.Workspace
	// Per-process env, not os.Setenv: os.Setenv is process-global and would
	// make concurrent sessions share (and clobber) one spool directory.
	cmd.Env = append(os.Environ(), conversation.SpoolEnvVar+"="+spoolDir)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	stdin, err := cmd.StdinPipe()
	if err != nil {
		sendError(ctx, conn, fmt.Sprintf("stdin pipe: %v", err))
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError(ctx, conn, fmt.Sprintf("stdout pipe: %v", err))
		return
	}

	if err := cmd.Start(); err != nil {
		sendError(ctx, conn, fmt.Sprintf("start claude: %v", err))
		return
	}

	w.mu.Lock()
	w.current = cmd.Process
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.current = nil
		w.mu.Unlock()
	}()

	// SIGTERM on shutdown
	go func() {
		<-ctx.Done()
		w.mu.Lock()
		if w.current != nil {
			w.current.Signal(syscall.SIGTERM)
		}
		w.mu.Unlock()
	}()

	// Write message to stdin
	text := msg.Text
	if msg.SystemPrompt != "" {
		text = msg.SystemPrompt + "\n\n---\n\nUser message:\n" + text
	}
	if paths := downloadAttachments(cfg, msg.Attachments); len(paths) > 0 {
		text += "\n\n[Attached files saved to workspace - use the Read tool to view them]\n"
		for _, p := range paths {
			text += fmt.Sprintf("  - %s\n", p)
		}
	}
	fmt.Fprint(stdin, text)
	stdin.Close()

	// sessionID tracks the initiating Claude session id as the worker learns
	// it: known up front for a resumed session (msg.SessionID), otherwise
	// captured from the earliest stream event that carries one (the initial
	// system/init event, well before the final result event). It is what
	// stamps any conversation_open frame drained from the spool — never a
	// session id supplied by the request file itself.
	sessionID := msg.SessionID
	send := func(ctx context.Context, frame map[string]any) error {
		return wsjson.Write(ctx, conn, frame)
	}

	// Stream stdout line by line
	var lastSessionID, lastResultText string
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		wsjson.Write(ctx, conn, map[string]any{"type": "stream", "data": line})

		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil {
			if sessionID == "" {
				if sid, ok := event["session_id"].(string); ok && sid != "" {
					sessionID = sid
				}
			}
			if event["type"] == "result" {
				lastSessionID, _ = event["session_id"].(string)
				lastResultText, _ = event["result"].(string)
			}
		}

		// Drain after every line so a request written mid-session is
		// emitted before the invocation returns, stamped with whatever
		// session id is known so far.
		drainSpool(ctx, spoolDir, sessionID, send)
	}

	waitErr := cmd.Wait()
	// Final drain after process exit, regardless of exit status, so a
	// request written just before the process exited is still emitted.
	drainSpool(ctx, spoolDir, sessionID, send)

	if waitErr != nil {
		detail := stderrBuf.String()
		if len(detail) > 2000 {
			detail = detail[:2000]
		}
		wsjson.Write(ctx, conn, map[string]any{
			"type":   "error",
			"error":  fmt.Sprintf("claude exited: %v", waitErr),
			"detail": detail,
		})
		log.Printf("Claude failed: %v", waitErr)
		return
	}

	wsjson.Write(ctx, conn, map[string]any{
		"type":             "result",
		"session_id":       lastSessionID,
		"agent_session_id": msg.AgentSessionID,
		"output":           lastResultText,
	})
	log.Printf("Claude finished: session=%s", lastSessionID)
}

func sendError(ctx context.Context, conn *websocket.Conn, detail string) {
	wsjson.Write(ctx, conn, map[string]any{
		"type":  "error",
		"error": detail,
	})
}
