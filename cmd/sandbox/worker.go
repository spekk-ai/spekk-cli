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
	defer pool.Release(w)

	for {
		select {
		case msg, ok := <-w.msgs:
			if !ok {
				return
			}
			w.invoke(ctx, cfg, conns, msg)
		default:
			// Ask the pool to release the slot. It refuses while a message
			// is queued, which closes the window where a dispatch lands
			// between this check and the release.
			if pool.finish(w) {
				return
			}
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
