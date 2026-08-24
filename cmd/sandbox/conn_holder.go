package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// finalSendTimeout bounds how long a frame that ends a turn waits for a live
// connection. It is longer than reconnectMax, so a single reconnect never
// loses a result.
const finalSendTimeout = 90 * time.Second

// errNoConnection reports that a frame could not be sent because no connection
// was live. It is not a protocol error: the connection drops several times a
// day for reasons outside this program.
var errNoConnection = errors.New("no live connection to the control host")

// connHolder publishes the connection that is live right now.
//
// A turn outlives the connection it started on, so a sender has to resolve the
// connection at the moment it sends rather than capture one when the turn
// began. A worker that held its own *websocket.Conn would still be holding the
// closed one after a reconnect, and its result would go nowhere.
type connHolder struct {
	mu      sync.Mutex
	conn    *websocket.Conn
	changed chan struct{}
}

func newConnHolder() *connHolder {
	return &connHolder{changed: make(chan struct{})}
}

// set publishes conn and wakes everything waiting for one.
func (h *connHolder) set(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conn = conn
	close(h.changed)
	h.changed = make(chan struct{})
}

// clear marks the holder as having no live connection. It does not wake
// waiters: they are waiting for a connection to arrive, not to depart.
func (h *connHolder) clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conn = nil
}

// current returns the live connection, which may be nil, together with a
// channel that closes when the next connection is published. Both come from
// one critical section, so a connection published between the two reads
// cannot be missed.
func (h *connHolder) current() (*websocket.Conn, <-chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.conn, h.changed
}

// send writes frame on the live connection, and drops it if there is none.
//
// This is the right behavior for a stream frame, which drives a live display:
// one that missed its moment has no value, and waiting here would stall the
// read of the child process's stdout.
func (h *connHolder) send(ctx context.Context, frame any) error {
	conn, _ := h.current()
	if conn == nil {
		return errNoConnection
	}
	return wsjson.Write(ctx, conn, frame)
}

// sendFinal writes a frame that ends a turn, waiting up to finalSendTimeout
// for a live connection.
//
// The control host acts on these frames, and a turn that reports nothing is
// indistinguishable from a turn that died. A reconnect takes seconds, so a
// bounded wait carries the report across it.
//
// The one deadline bounds the writing as well as the waiting. A write to a
// connection that is dead but not yet detected can block for as long as the
// kernel takes to give up on it, which is far longer than a reconnect, so a
// write with no deadline could hold the frame past every chance to deliver it.
func (h *connHolder) sendFinal(ctx context.Context, frame any) error {
	ctx, cancel := context.WithTimeout(ctx, finalSendTimeout)
	defer cancel()

	for {
		conn, changed := h.current()
		if conn != nil {
			if err := wsjson.Write(ctx, conn, frame); err == nil {
				return nil
			}
			// The connection died under this write. changed was read in the
			// same critical section as conn, so if the client has already
			// published a replacement it is closed and the wait below returns
			// at once.
		}

		select {
		case <-changed:
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", errNoConnection, ctx.Err())
		}
	}
}
