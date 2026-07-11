package main

import (
	"context"
	"fmt"
	"log"
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
	return fmt.Sprintf("%s://%s/ws/agent/%s/", scheme, c.cfg.Host, c.cfg.Token)
}

func (c *AgentClient) Run(ctx context.Context) {
	delay := reconnectBase
	for {
		err := c.connect(ctx)
		if ctx.Err() != nil {
			return // clean shutdown
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
	opts := &websocket.DialOptions{
		CompressionMode: websocket.CompressionDisabled,
	}
	conn, _, err := websocket.Dial(ctx, c.wsURL(), opts)
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

		switch msg.Type {
		case MessageTypeMessage:
			c.handleMessage(ctx, conn, msg)
		case MessageTypeCancel:
			c.handleCancel(msg)
		case MessageTypeHeartbeatAck:
			// ignore
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
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
