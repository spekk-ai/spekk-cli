package main

import (
	"context"
	"os"
	"sync"
	"syscall"

	"github.com/coder/websocket"
)

type Worker struct {
	agentSessionID string
	msgs           chan Message
	mu             sync.Mutex
	current        *os.Process
}

func (w *Worker) Run(ctx context.Context, cfg Config, conn *websocket.Conn, pool *WorkerPool) {
	defer pool.Release(w.agentSessionID)

	for {
		select {
		case msg, ok := <-w.msgs:
			if !ok {
				return
			}
			w.invoke(ctx, cfg, conn, msg)
		default:
			return // queue empty - release
		}
	}
}

func (w *Worker) Cancel() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != nil {
		w.current.Signal(syscall.SIGTERM)
	}
}
