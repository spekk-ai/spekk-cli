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
	// cancelled records a cancel that reached the turn in flight. invoke
	// reads it under mu before it starts the process, so a cancel that lands
	// before the process exists still stops the turn. next clears it when
	// the runner takes a message, so it never reaches a later turn.
	cancelled bool
}

// Run drains this worker's queue. ctx is the process lifetime, so a turn
// outlives the connection that carried its dispatch, and conns resolves
// whichever connection is live at the moment a frame is sent.
func (w *Worker) Run(ctx context.Context, cfg Config, conns *connHolder, pool *WorkerPool) {
	defer pool.Release(w)

	for {
		if msg, ok := w.next(); ok {
			w.invoke(ctx, cfg, conns, msg)
			continue
		}
		// Ask the pool to release the slot. It refuses while a message is
		// queued, which closes the window where a dispatch lands between
		// this check and the release.
		if pool.finish(w) {
			return
		}
	}
}

// next takes the next queued message, if there is one. It takes it under
// mu, the mutex Cancel holds while it empties the queue, so each message
// that was queued before a cancel is either removed by that cancel or is
// the turn in flight that the cancel marks.
func (w *Worker) next() (Message, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case msg := <-w.msgs:
		w.cancelled = false
		return msg, true
	default:
		return Message{}, false
	}
}

// Cancel stops the session: it removes every queued message, marks the turn
// in flight as cancelled, and signals its process if one runs. It returns
// the removed messages, because the control host opened a turn for each of
// them and waits for a frame that ends it.
func (w *Worker) Cancel() []Message {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Only next and Cancel receive from msgs, and both hold mu, so a queue
	// with a length is a queue that a receive cannot block on.
	var removed []Message
	for len(w.msgs) > 0 {
		removed = append(removed, <-w.msgs)
	}
	w.cancelled = true
	if w.current != nil {
		w.current.Signal(syscall.SIGTERM)
	}
	return removed
}
