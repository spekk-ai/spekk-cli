package main

import (
	"context"
	"os"
	"sync"
	"syscall"
)

type Worker struct {
	agentSessionID string
	msgs           chan Message
	mu             sync.Mutex
	current        *os.Process
}

// Run drains this worker's queue. ctx is the process lifetime, so a turn
// outlives the connection that carried its dispatch, and conns resolves
// whichever connection is live at the moment a frame is sent.
func (w *Worker) Run(ctx context.Context, cfg Config, conns *connHolder, pool *WorkerPool) {
	defer pool.Release(w.agentSessionID)

	for {
		select {
		case msg, ok := <-w.msgs:
			if !ok {
				return
			}
			w.invoke(ctx, cfg, conns, msg)
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
