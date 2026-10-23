package main

import (
	"testing"
	"time"
)

// --- Worker routing ---

// TestWorkerRouting verifies that messages for the same agent_session_id
// reach the same worker, and that only the first dispatch asks for a runner.
// A second runner over one worker released the same slot twice and wedged
// the client.
func TestWorkerRouting(t *testing.T) {
	pool := NewWorkerPool(5)

	msg1 := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}
	msg2 := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}

	w1, accepted := pool.Dispatch(msg1)
	if !accepted || w1 == nil {
		t.Fatalf("first dispatch: worker %v, accepted %v", w1, accepted)
	}

	w2, accepted := pool.Dispatch(msg2)
	if !accepted {
		t.Fatal("a follow-up for a running session must be accepted")
	}
	if w2 != nil {
		t.Error("a follow-up must not ask for a second runner")
	}
	if len(w1.msgs) != 2 {
		t.Errorf("both messages must queue on the one worker, got %d", len(w1.msgs))
	}
}

// TestWorkerRoutingDistinctSessions verifies that distinct session IDs
// get different workers.
func TestWorkerRoutingDistinctSessions(t *testing.T) {
	pool := NewWorkerPool(5)

	msgA := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}
	msgB := Message{Type: MessageTypeMessage, AgentSessionID: "session-B"}

	wA, _ := pool.Dispatch(msgA)
	if wA == nil {
		t.Fatal("expected worker for session-A")
	}
	wB, _ := pool.Dispatch(msgB)
	if wB == nil {
		t.Fatal("expected worker for session-B")
	}

	if wA == wB {
		t.Errorf("expected different workers for different sessions, got the same")
	}
}

// --- Session cap ---

// TestSessionCap verifies that when all 5 worker slots are occupied by
// distinct sessions, a 6th new session gets nil (capacity exceeded).
func TestSessionCap(t *testing.T) {
	pool := NewWorkerPool(5)

	for i := 0; i < 5; i++ {
		msg := Message{
			Type:           MessageTypeMessage,
			AgentSessionID: "session-" + string(rune('A'+i)),
		}
		w, _ := pool.Dispatch(msg)
		if w == nil {
			t.Fatalf("expected worker for session %d, got nil", i)
		}
	}

	// 6th new session should be rejected
	msg6 := Message{Type: MessageTypeMessage, AgentSessionID: "session-F"}
	w6, accepted := pool.Dispatch(msg6)
	if accepted || w6 != nil {
		t.Errorf("6th session must be refused, got worker %v accepted %v", w6, accepted)
	}
}

// TestFollowUpDoesNotWedgeThePool pins the failure a second runner caused.
// Two runners over one worker each released the same slot; the second
// release blocked on a channel already at its cap while holding the mutex,
// and every later dispatch then blocked on that mutex forever. The
// connection stayed up and the heartbeats kept flowing, so the sandbox
// looked healthy while accepting no work at all.
func TestFollowUpDoesNotWedgeThePool(t *testing.T) {
	pool := NewWorkerPool(2)

	w, _ := pool.Dispatch(Message{Type: MessageTypeMessage, AgentSessionID: "s"})
	if w == nil {
		t.Fatal("expected a worker")
	}
	// A follow-up arrives while the turn runs.
	pool.Dispatch(Message{Type: MessageTypeMessage, AgentSessionID: "s"})

	// Drain both messages the way a runner does, then let it finish.
	<-w.msgs
	<-w.msgs
	if !pool.finish(w) {
		t.Fatal("finish must release a worker whose queue is empty")
	}
	// A second release must be a no-op, not an extra token.
	pool.Release(w)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 2; i++ {
			if _, accepted := pool.Dispatch(Message{
				Type:           MessageTypeMessage,
				AgentSessionID: "later-" + string(rune('a'+i)),
			}); !accepted {
				t.Errorf("dispatch %d refused: the slot was not returned", i)
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch wedged: the pool never recovered the slot")
	}
}

// TestFinishKeepsAWorkerWithAQueuedMessage pins the handoff. A runner that
// checked its queue outside the lock could stop while Dispatch was
// enqueueing, and that message would wait on a worker nobody was draining.
func TestFinishKeepsAWorkerWithAQueuedMessage(t *testing.T) {
	pool := NewWorkerPool(2)
	w, _ := pool.Dispatch(Message{Type: MessageTypeMessage, AgentSessionID: "s"})
	if pool.finish(w) {
		t.Fatal("finish must refuse while a message is queued")
	}
	<-w.msgs
	if !pool.finish(w) {
		t.Fatal("finish must release once the queue is empty")
	}
}

// TestFullSessionQueueIsRefusedNotBlocked pins the other way one dispatch
// could stall every other one. The queue send happens under the pool mutex,
// so a blocking send on a full queue holds that mutex until the worker
// drains — and no other session can be dispatched meanwhile.
func TestFullSessionQueueIsRefusedNotBlocked(t *testing.T) {
	pool := NewWorkerPool(2)
	msg := Message{Type: MessageTypeMessage, AgentSessionID: "s"}
	if _, accepted := pool.Dispatch(msg); !accepted {
		t.Fatal("first dispatch must be accepted")
	}
	// Fill the queue. Derive the room from the channel rather than a
	// literal, so a capacity change cannot silently weaken this test.
	room := cap(poolQueue(pool, "s")) - len(poolQueue(pool, "s"))
	for i := 0; i < room; i++ {
		if _, accepted := pool.Dispatch(msg); !accepted {
			t.Fatalf("follow-up %d must be accepted while the queue has room", i)
		}
	}

	done := make(chan bool, 1)
	go func() {
		_, accepted := pool.Dispatch(msg)
		done <- accepted
	}()
	select {
	case accepted := <-done:
		if accepted {
			t.Error("a full queue must be refused, not accepted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch blocked on a full queue while holding the pool mutex")
	}

	// The pool still works for another session.
	if _, accepted := pool.Dispatch(Message{Type: MessageTypeMessage, AgentSessionID: "other"}); !accepted {
		t.Error("an unrelated session must still be dispatchable")
	}
}

// poolQueue returns a session's queue, so a test can derive its room
// instead of restating the capacity the pool chose.
func poolQueue(p *WorkerPool, agentSessionID string) chan Message {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sessions[agentSessionID].msgs
}
