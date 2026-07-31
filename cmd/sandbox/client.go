package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	heartbeatInterval = 30 * time.Second
	reconnectBase     = 3 * time.Second
	reconnectMax      = 60 * time.Second
	wsMaxMessageSize  = 20 * 1024 * 1024 // 20MB
)

type AgentClient struct {
	cfg  Config
	pool *WorkerPool
}

func NewAgentClient(cfg Config) *AgentClient {
	return &AgentClient{
		cfg:  cfg,
		pool: NewWorkerPool(5),
	}
}

func (c *AgentClient) wsURL() string {
	scheme := "wss"
	if contains(c.cfg.Host, "localhost") {
		scheme = "ws"
	}
	// Send the token in both the URL path and the Authorization header
	// (see dialOptions) for compatibility.
	return fmt.Sprintf("%s://%s/ws/agent/%s/", scheme, c.cfg.Host, c.cfg.Token)
}

// dialOptions builds the websocket.DialOptions used to connect. The agent
// token is sent as an Authorization header in addition to (not instead of)
// the path token wsURL() embeds — see the comment on wsURL(). Split out from
// connect() so the header construction is exercisable without a real dial.
func (c *AgentClient) dialOptions() *websocket.DialOptions {
	return &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
		HTTPHeader: http.Header{
			"Authorization":    []string{"Bearer " + c.cfg.Token},
			protocolHeaderName: []string{ProtocolVersion},
		},
	}
}

func (c *AgentClient) Run(ctx context.Context) {
	delay := reconnectBase
	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return // clean shutdown
		}
		if isProtocolReject(err) {
			log.Printf("Control host rejected protocol version %s (close 4004). Update this sandbox's agent-client.", ProtocolVersion)
		}
		log.Printf("Connection lost: %v. Reconnecting in %s...", err, delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, reconnectMax)
	}
}

func (c *AgentClient) connect(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, c.wsURL(), c.dialOptions())
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(wsMaxMessageSize)

	log.Println("Connected")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.heartbeat(ctx, conn)

	return c.readLoop(ctx, conn)
}

func (c *AgentClient) heartbeat(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := wsjson.Write(ctx, conn, map[string]string{"type": "heartbeat"})
			if err != nil {
				return
			}
		}
	}
}

func (c *AgentClient) readLoop(ctx context.Context, conn *websocket.Conn) error {
	for {
		var msg Message
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			return err
		}

		c.handleInbound(ctx, conn, msg)
	}
}

// handleInbound dispatches a single decoded inbound frame. Split out from
// readLoop so the dispatch logic is exercisable in tests without a real
// websocket connection.
func (c *AgentClient) handleInbound(ctx context.Context, conn *websocket.Conn, msg Message) {
	switch msg.Type {
	case MessageTypeMessage:
		c.handleMessage(ctx, conn, msg)
	case MessageTypeCancel:
		c.handleCancel(msg)
	case MessageTypeHeartbeatAck:
		// ignore
	case MessageTypeError:
		handleErrorFrame(msg)
	case MessageTypeWelcome:
		c.handleWelcome(msg)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// conversationOpenErrorCodes are the error codes the control host sends in
// reply to a rejected conversation_open request. Frames carrying one of
// these are logged as conversation_open rejections rather than generic
// errors.
var conversationOpenErrorCodes = map[string]bool{
	"conversation_open_invalid":    true,
	"conversation_open_no_channel": true,
	"conversation_open_failed":     true,
}

// handleErrorFrame logs an inbound "error" frame legibly. It never tears
// down the connection or the worker — receiving an error frame is a
// non-fatal event, on par with any other frame this loop handles.
func handleErrorFrame(msg Message) {
	if conversationOpenErrorCodes[msg.Error] {
		log.Printf("conversation_open rejected: %s — %s", msg.Error, msg.Detail)
		return
	}
	if msg.Error == "" && msg.Detail == "" {
		// A control host predating the typed error contract sends error
		// frames without code/detail fields; say so rather than logging an
		// empty "—".
		log.Printf("error frame received with no code/detail (control host may predate the typed error contract)")
		return
	}
	log.Printf("error frame received: %s — %s", msg.Error, msg.Detail)
}

func (c *AgentClient) handleMessage(ctx context.Context, conn *websocket.Conn, msg Message) {
	w := c.pool.Dispatch(msg)
	if w == nil {
		wsjson.Write(ctx, conn, map[string]any{
			"type":   "error",
			"error":  "capacity_exceeded",
			"detail": "All 5 agent worker slots are busy. Try again shortly.",
		})
		return
	}

	go w.Run(ctx, c.cfg, conn, c.pool)
}

func (c *AgentClient) handleCancel(msg Message) {
	c.pool.Cancel(msg.AgentSessionID)
}

type Config struct {
	Token     string
	Host      string
	Workspace string
}

func loadConfig() Config {
	token := os.Getenv("SPEKK_AGENT_TOKEN")
	host := os.Getenv("SPEKK_HOST")
	workspace := os.Getenv("WORKSPACE")

	if token == "" || host == "" {
		log.Fatal("SPEKK_AGENT_TOKEN and SPEKK_HOST must be set")
	}
	if workspace == "" {
		workspace = "/opt/spekk/workspace"
	}

	return Config{
		Token:     token,
		Host:      host,
		Workspace: workspace,
	}
}

// handleWelcome checks the control host's advertised protocol version. The
// server owns enforcement; the client only informs the operator, so a
// mismatch warns and the connection continues.
func (c *AgentClient) handleWelcome(msg Message) {
	if protocolMajor(msg.Protocol) == protocolMajor(ProtocolVersion) {
		log.Printf("Control host protocol %s (client %s)", msg.Protocol, ProtocolVersion)
		return
	}
	log.Printf(
		"WARNING: control host speaks protocol %s, this client speaks %s. Update this sandbox's agent-client.",
		msg.Protocol, ProtocolVersion,
	)
}
