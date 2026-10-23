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
	// dropped holds the queued messages that a cancel removed, in the order
	// they were received. The control host ends the oldest open turn of a
	// session for each frame that ends a turn, so the runner ends their
	// turns only after the turn in flight sent its own final frame.
	dropped []Message
}

// Run drains this worker's queue. ctx is the process lifetime, so a turn
// outlives the connection that carried its dispatch, and conns resolves
// whichever connection is live at the moment a frame is sent.
func (w *Worker) Run(ctx context.Context, cfg Config, conns *connHolder, pool *WorkerPool) {
	defer pool.Release(w)

	for {
		w.endDropped(ctx, conns)
		afterEndDropped()
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

// afterEndDropped runs between endDropped and next. It does nothing, and a
// test replaces it to hold the runner in the window where a cancel and a new
// message can arrive between the two.
var afterEndDropped = func() {}

// next takes the next queued message, if there is one. It takes it under
// mu, the mutex Cancel holds while it empties the queue, so each message
// that was queued before a cancel is either removed by that cancel or is
// the turn in flight that the cancel marks. It takes no message while a
// cancel's removed messages wait for their frames: a cancel can land after
// endDropped released mu, and the loop must end those turns first.
func (w *Worker) next() (Message, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.dropped) > 0 {
		return Message{}, false
	}
	select {
	case msg := <-w.msgs:
		w.cancelled = false
		return msg, true
	default:
		return Message{}, false
	}
}

// endDropped ends the turn of each message that a cancel removed, in the
// order the messages were received. The runner calls it between turns, so
// these frames go after the final frame of the turn in flight and before
// any frame of a later turn.
func (w *Worker) endDropped(ctx context.Context, conns *connHolder) {
	w.mu.Lock()
	dropped := w.dropped
	w.dropped = nil
	w.mu.Unlock()
	for _, m := range dropped {
		sendError(ctx, conns, m.AgentSessionID, cancelledBeforeStartDetail)
	}
}

// hasDropped reports whether a cancel removed a message whose turn the
// runner has not ended yet.
func (w *Worker) hasDropped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.dropped) > 0
}

// Cancel stops the session: it removes every queued message, marks the turn
// in flight as cancelled, and signals its process if one runs. The control
// host opened a turn for each removed message and waits for a frame that
// ends it, so Cancel keeps them for the runner to end. It returns how many
// it removed.
func (w *Worker) Cancel() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Only next and Cancel receive from msgs, and both hold mu, so a queue
	// with a length is a queue that a receive cannot block on.
	n := len(w.msgs)
	for len(w.msgs) > 0 {
		w.dropped = append(w.dropped, <-w.msgs)
	}
	w.cancelled = true
	if w.current != nil {
		w.current.Signal(syscall.SIGTERM)
	}
	return n
}
