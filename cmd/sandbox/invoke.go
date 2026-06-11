package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"syscall"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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

	cmd := exec.Command("claude", args...)
	cmd.Dir = cfg.Workspace

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
		if json.Unmarshal([]byte(line), &event) == nil && event["type"] == "result" {
			lastSessionID, _ = event["session_id"].(string)
			lastResultText, _ = event["result"].(string)
		}
	}

	if err := cmd.Wait(); err != nil {
		detail := stderrBuf.String()
		if len(detail) > 2000 {
			detail = detail[:2000]
		}
		wsjson.Write(ctx, conn, map[string]any{
			"type":   "error",
			"error":  fmt.Sprintf("claude exited: %v", err),
			"detail": detail,
		})
		log.Printf("Claude failed: %v", err)
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
