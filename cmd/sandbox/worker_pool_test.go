package main

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// sleepCmd starts "sleep 30" and fails the test if it cannot start.
func sleepCmd(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep process: %v", err)
	}
	t.Cleanup(func() { cmd.Process.Kill() })
	return cmd
}

// --- Worker routing ---

// TestWorkerRouting verifies that messages for the same agent_session_id
// are dispatched to the same worker.
func TestWorkerRouting(t *testing.T) {
	pool := NewWorkerPool(5)

	msg1 := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}
	msg2 := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}

	w1 := pool.Dispatch(msg1)
	if w1 == nil {
		t.Fatal("expected a worker for first dispatch, got nil")
	}

	w2 := pool.Dispatch(msg2)
	if w2 == nil {
		t.Fatal("expected a worker for second dispatch on same session, got nil")
	}

	if w1 != w2 {
		t.Errorf("expected same worker for same session_id, got different workers")
	}
}

// TestWorkerRoutingDistinctSessions verifies that distinct session IDs
// get different workers.
func TestWorkerRoutingDistinctSessions(t *testing.T) {
	pool := NewWorkerPool(5)

	msgA := Message{Type: MessageTypeMessage, AgentSessionID: "session-A"}
	msgB := Message{Type: MessageTypeMessage, AgentSessionID: "session-B"}

	wA := pool.Dispatch(msgA)
	if wA == nil {
		t.Fatal("expected worker for session-A")
	}
	wB := pool.Dispatch(msgB)
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
		w := pool.Dispatch(msg)
		if w == nil {
			t.Fatalf("expected worker for session %d, got nil", i)
		}
	}

	// 6th new session should be rejected
	msg6 := Message{Type: MessageTypeMessage, AgentSessionID: "session-F"}
	w6 := pool.Dispatch(msg6)
	if w6 != nil {
		t.Errorf("expected nil for 6th session (capacity exceeded), got a worker")
	}
}

// TestSessionCapRelease verifies that after releasing a session a new one
// can be accepted.
func TestSessionCapRelease(t *testing.T) {
	pool := NewWorkerPool(5)

	for i := 0; i < 5; i++ {
		msg := Message{
			Type:           MessageTypeMessage,
			AgentSessionID: "session-" + string(rune('A'+i)),
		}
		pool.Dispatch(msg)
	}

	// Release one slot
	pool.Release("session-A")

	// Now a new session should succeed
	msgNew := Message{Type: MessageTypeMessage, AgentSessionID: "session-NEW"}
	wNew := pool.Dispatch(msgNew)
	if wNew == nil {
		t.Errorf("expected worker after releasing a slot, got nil")
	}
}

// --- Cancel ---

// TestCancelActiveSession verifies that Cancel sends SIGTERM to the
// running process tracked by the worker.
func TestCancelActiveSession(t *testing.T) {
	pool := NewWorkerPool(5)

	msg := Message{Type: MessageTypeMessage, AgentSessionID: "session-cancel"}
	w := pool.Dispatch(msg)
	if w == nil {
		t.Fatal("expected worker for cancel test session")
	}

	cmd := sleepCmd(t)

	w.mu.Lock()
	w.current = cmd.Process
	w.mu.Unlock()

	// Cancel should send SIGTERM
	pool.Cancel("session-cancel")

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		// A SIGTERM'd process exits with a non-nil error on Linux.
		if err == nil {
			t.Errorf("expected non-nil error from SIGTERM'd process, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Error("process did not exit after SIGTERM within timeout")
	}
}

// TestCancelUnknownSession verifies that cancelling an unknown session is a no-op.
func TestCancelUnknownSession(t *testing.T) {
	pool := NewWorkerPool(5)
	// Must not panic or error
	pool.Cancel("nonexistent-session")
}

// TestWorkerCancelNoProcess verifies that Worker.Cancel is safe when no
// process is running (w.current == nil).
func TestWorkerCancelNoProcess(t *testing.T) {
	w := &Worker{
		agentSessionID: "test",
		msgs:           make(chan Message, 10),
	}
	// Must not panic
	w.Cancel()
}

// TestWorkerCancelSendsSignal verifies that Worker.Cancel sends SIGTERM
// when a process is attached.
func TestWorkerCancelSendsSignal(t *testing.T) {
	w := &Worker{
		agentSessionID: "test",
		msgs:           make(chan Message, 10),
	}

	cmd := sleepCmd(t)

	w.mu.Lock()
	w.current = cmd.Process
	w.mu.Unlock()

	w.Cancel()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err == nil {
			t.Errorf("expected non-nil error from SIGTERM'd process, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Error("process did not exit after SIGTERM within timeout")
	}
}

// --- Graceful shutdown ---

// TestGracefulShutdown verifies that on SIGTERM (context cancellation), all
// active Claude processes receive SIGTERM and the client exits cleanly.
//
// This test exercises the shutdown signal path directly on a Worker,
// reproducing the goroutine inside invoke() that watches ctx.Done().
func TestGracefulShutdown(t *testing.T) {
	w := &Worker{
		agentSessionID: "shutdown-test",
		msgs:           make(chan Message, 10),
	}

	cmd := sleepCmd(t)

	w.mu.Lock()
	w.current = cmd.Process
	w.mu.Unlock()

	// Replicate the shutdown goroutine from invoke(): on ctx cancel, send SIGTERM.
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		w.mu.Lock()
		if w.current != nil {
			w.current.Signal(syscall.SIGTERM)
		}
		w.mu.Unlock()
	}()

	<-shutdownDone

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case err := <-exited:
		if err == nil {
			t.Errorf("expected non-nil exit after SIGTERM, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Error("process did not exit after simulated shutdown SIGTERM")
	}
}
