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
	"github.com/spekk-ai/spekk-cli/internal/agentconn"
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
	terms terminals
	// record is read and written only by Run's goroutine. The zero value
	// keeps no record.
	record connectionRecorder

	// procCtx is the process lifetime, handed to Run by main. Workers get this
	// context and never the per-connection one: a turn belongs to the process,
	// not to the connection that happened to carry its dispatch.
	procCtx context.Context
}

func NewAgentClient(cfg Config) *AgentClient {
	return &AgentClient{
		cfg:     cfg,
		pool:    NewWorkerPool(numWorkers),
		conns:   newConnHolder(),
		record:  newConnectionRecorder(cfg.ConnectionRecord),
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

// establishedMin is how long a connection must live before it counts as
// established and resets the backoff. A connection that ends sooner tells us
// the control host is not ready, whatever the dial reported.
const establishedMin = 10 * time.Second

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
//
// Established means the connection lived, not merely that the dial returned.
// A control host that accepts and closes at once — which is exactly the
// documented protocol reject, close 4004 — otherwise reset the backoff on
// every attempt, and an outdated sandbox dialed it twenty times a minute for
// the length of the deploy gap.
func (c *AgentClient) connect(ctx context.Context) (established bool, err error) {
	conn, resp, err := websocket.Dial(ctx, c.wsURL(), c.dialOptions())
	if err != nil {
		// A dial that a shutdown stopped says nothing about the control host.
		if ctx.Err() == nil {
			c.record.failed(time.Now(), resp, err)
		}
		return false, fmt.Errorf("dial: %w", err)
	}
	opened := time.Now()
	c.record.connected(opened)
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

	err = c.readLoop(ctx, conn)
	return time.Since(opened) >= establishedMin, err
}

func (c *AgentClient) heartbeat(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := wsjson.Write(ctx, conn, heartbeatFrame())
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
	case MessageTypeTerminalOpen, MessageTypeTerminalInput, MessageTypeTerminalResize, MessageTypeTerminalClose:
		c.handleTerminal(msg)
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
	w, accepted := c.pool.Dispatch(msg)
	if !accepted {
		// An immediate refusal, so it belongs to the connection that asked.
		c.conns.send(ctx, map[string]any{
			"type":             "error",
			"error":            "capacity_exceeded",
			"detail":           "No agent worker slot is free, or this session's queue is full. Try again shortly.",
			"agent_session_id": msg.AgentSessionID,
		})
		return
	}

	// Nil means the message joined a worker that is already draining, and
	// that worker's own runner will take it. Starting a second runner over
	// one worker wedged the client.
	if w == nil {
		return
	}

	// The process context, never the connection's. A dropped connection used
	// to cancel this context, which signaled the running claude process and
	// killed the turn; the client then reconnected onto work that was already
	// dead. The worker resolves its connection through c.conns as it sends.
	go w.Run(c.procCtx, c.cfg, c.conns, c.pool)
}

// cancelledBeforeStartDetail ends a turn that a cancel stopped before its
// process started.
const cancelledBeforeStartDetail = "Cancelled before the turn started."

// handleCancel stops the session, and ends the turn of each queued message
// it removed. The control host opened a turn for each dispatch and closes
// one for each frame that ends a turn, so a message dropped in silence would
// leave its turn open. The frames wait for a connection, so they go out on a
// goroutine and the read loop does not wait for them.
func (c *AgentClient) handleCancel(msg Message) {
	removed := c.pool.Cancel(msg.AgentSessionID)
	if len(removed) == 0 {
		return
	}
	go func() {
		for _, m := range removed {
			sendError(c.procCtx, c.conns, m.AgentSessionID, cancelledBeforeStartDetail)
		}
	}()
}

type Config struct {
	Token     string
	Host      string
	Workspace string
	AgentID   string
	// ConnectionRecord is the file of the handshake record. Empty keeps no
	// record.
	ConnectionRecord string
}

func loadConfig() Config {
	token := os.Getenv("SPEKK_AGENT_TOKEN")
	host := os.Getenv("SPEKK_HOST")
	workspace := os.Getenv("WORKSPACE")
	agentID := os.Getenv("SPEKK_AGENT_ID")

	if token == "" || host == "" {
		log.Fatal("SPEKK_AGENT_TOKEN and SPEKK_HOST must be set")
	}
	if workspace == "" {
		workspace = "/opt/spekk/workspace"
	}

	return Config{
		Token:            token,
		Host:             host,
		Workspace:        workspace,
		AgentID:          agentID,
		ConnectionRecord: agentconn.Path,
	}
}

// handleWelcome checks the control host's advertised protocol version. The
// server owns enforcement; the client only informs the operator, so a
// mismatch warns and the connection continues.
func (c *AgentClient) handleWelcome(msg Message) {
	if c.conns != nil {
		c.conns.setCapabilities(msg.Capabilities)
	}
	if protocolMajor(msg.Protocol) == protocolMajor(ProtocolVersion) {
		log.Printf("Control host protocol %s (client %s)", msg.Protocol, ProtocolVersion)
		return
	}
	log.Printf(
		"WARNING: control host speaks protocol %s, this client speaks %s. Update this sandbox's agent-client.",
		msg.Protocol, ProtocolVersion,
	)
}
