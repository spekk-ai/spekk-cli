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
	cfg   Config
	pool  *WorkerPool
	conns *connHolder

	// procCtx is the process lifetime, handed to Run by main. Workers get this
	// context and never the per-connection one: a turn belongs to the process,
	// not to the connection that happened to carry its dispatch.
	procCtx context.Context
}

func NewAgentClient(cfg Config) *AgentClient {
	return &AgentClient{
		cfg:     cfg,
		pool:    NewWorkerPool(5),
		conns:   newConnHolder(),
		procCtx: context.Background(),
	}
}

// wsURL builds the dial target. The token is deliberately absent: it travels
// only in the Authorization header (see dialOptions). A token in the URL path
// leaks into access logs, proxy logs, and any error string that echoes the
// target -- including this client's own reconnect log.
func (c *AgentClient) wsURL() string {
	scheme := "wss"
	if contains(c.cfg.Host, "localhost") {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s/ws/agent/", scheme, c.cfg.Host)
}

// dialOptions builds the websocket.DialOptions used to connect. The
// Authorization header is the sole carrier of the agent token. Split out from
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
	c.procCtx = ctx

	var delay time.Duration
	for {
		established, err := c.connect(ctx)
		if ctx.Err() != nil {
			return // clean shutdown
		}
		if isProtocolReject(err) {
			log.Printf("Control host rejected protocol version %s (close 4004). Update this sandbox's agent-client.", ProtocolVersion)
		}
		delay = reconnectDelay(delay, established)
		log.Printf("Connection lost: %v. Reconnecting in %s...", err, delay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// reconnectDelay returns how long to wait before the next dial attempt, given
// the previous wait and whether the attempt that just ended had actually
// established a connection.
//
// An established connection resets the backoff. The delay measures how long
// the control host has been unreachable, not how long this process has been
// alive: while the delay only ever grew, a process that had dropped about five
// times waited the full reconnectMax for every later reconnect, even when the
// connection before it had been healthy for hours. A zero last, before the
// first attempt, also yields reconnectBase.
func reconnectDelay(last time.Duration, established bool) time.Duration {
	if established || last == 0 {
		return reconnectBase
	}
	return min(last*2, reconnectMax)
}

// connect dials, serves the connection until it ends, and reports whether the
// connection was ever established. A dial that failed must keep backing off,
// so that a control host which is down is not dialed every few seconds.
func (c *AgentClient) connect(ctx context.Context) (established bool, err error) {
	conn, _, err := websocket.Dial(ctx, c.wsURL(), c.dialOptions())
	if err != nil {
		return false, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(wsMaxMessageSize)

	log.Println("Connected")

	// Publish before serving and withdraw after, so a worker sending mid-turn
	// resolves this connection while it lives and waits for the next one after
	// it ends.
	c.conns.set(conn)
	defer c.conns.clear()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go c.heartbeat(ctx, conn)

	return true, c.readLoop(ctx, conn)
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

		c.handleInbound(ctx, msg)
	}
}

// handleInbound dispatches a single decoded inbound frame. Split out from
// readLoop so the dispatch logic is exercisable in tests without a real
// websocket connection.
//
// ctx here is the per-connection context, and it is used only for an immediate
// reply to the frame being handled. Work that outlives the frame takes
// c.procCtx instead — see handleMessage.
func (c *AgentClient) handleInbound(ctx context.Context, msg Message) {
	switch msg.Type {
	case MessageTypeMessage:
		c.handleMessage(ctx, msg)
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

func (c *AgentClient) handleMessage(ctx context.Context, msg Message) {
	w := c.pool.Dispatch(msg)
	if w == nil {
		// An immediate refusal, so it belongs to the connection that asked.
		c.conns.send(ctx, map[string]any{
			"type":   "error",
			"error":  "capacity_exceeded",
			"detail": "All 5 agent worker slots are busy. Try again shortly.",
		})
		return
	}

	// The process context, never the connection's. A dropped connection used
	// to cancel this context, which signaled the running claude process and
	// killed the turn; the client then reconnected onto work that was already
	// dead. The worker resolves its connection through c.conns as it sends.
	go w.Run(c.procCtx, c.cfg, c.conns, c.pool)
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
