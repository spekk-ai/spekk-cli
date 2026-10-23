package main

import (
	"context"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// A cancel used to signal only the running process. The runner then took the
// next queued message and started a new turn, and a cancel that landed before
// the process started did nothing at all (specs/cancel-stops-the-session/).
// These tests drive the real dispatch path with the fake claude of
// reconnect_test.go.

const cancelSession = "agent-session-1" // the session dispatchTurn names

// startCancelClient connects a client to a test control host and returns the
// client and its first connection.
func startCancelClient(t *testing.T) (*AgentClient, *testControlHost, *websocket.Conn) {
	t.Helper()
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go client.Run(ctx)
	return client, host, host.nextConn(t)
}

// queueFollowUps dispatches n more messages for the session while its first
// turn runs, and waits until all of them are in the worker's queue.
func queueFollowUps(t *testing.T, client *AgentClient, conn *websocket.Conn, n int) {
	t.Helper()
	for range n {
		dispatchTurn(t, conn)
	}
	waitUntil(t, 20*time.Second, "the follow-ups never reached the queue", func() bool {
		client.pool.mu.Lock()
		defer client.pool.mu.Unlock()
		w := client.pool.sessions[cancelSession]
		return w != nil && len(w.msgs) == n
	})
}

// waitForRelease waits until the pool releases the session's worker. After
// that no turn of the session can start, so a count of runs is final.
func waitForRelease(t *testing.T, client *AgentClient) {
	t.Helper()
	waitUntil(t, 20*time.Second, "the pool never released the worker", func() bool {
		client.pool.mu.Lock()
		defer client.pool.mu.Unlock()
		return client.pool.sessions[cancelSession] == nil
	})
}

// runCount is how many turns the fake claude ran.
func runCount(t *testing.T, m turnMarkers) int {
	t.Helper()
	data, err := os.ReadFile(m.runs)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("read the run log: %v", err)
	}
	return strings.Count(string(data), "\n")
}

// awaitSessionErrors waits for n error frames and fails if one does not name
// the session. Each frame ends one turn that the control host opened.
func awaitSessionErrors(t *testing.T, host *testControlHost, n int) []map[string]any {
	t.Helper()
	var frames []map[string]any
	for range n {
		got := host.awaitFrame(t, MessageTypeError, 20*time.Second)
		if got.frame["agent_session_id"] != cancelSession {
			t.Errorf("error frame agent_session_id = %v, want %s", got.frame["agent_session_id"], cancelSession)
		}
		frames = append(frames, got.frame)
	}
	return frames
}

// TestCancelStopsTheQueuedTurns is the first fault of issue #212: after a
// cancel, the turns queued behind the running one still ran. The fake claude
// waits for a release file that this test never writes, so a queued turn
// that started would hold the worker and the pool would never release it.
func TestCancelStopsTheQueuedTurns(t *testing.T) {
	markers := fakeClaude(t)
	client, host, conn := startCancelClient(t)

	dispatchTurn(t, conn)
	waitForFile(t, markers.started, 20*time.Second, "the fake claude never started")
	queueFollowUps(t, client, conn, 2)

	if err := wsjson.Write(context.Background(), conn, map[string]any{
		"type":             MessageTypeCancel,
		"agent_session_id": cancelSession,
	}); err != nil {
		t.Fatalf("send cancel: %v", err)
	}

	// One frame for the signaled turn and one for each removed message. The
	// control host ends the oldest open turn for each frame, so the signaled
	// turn's frame must come first.
	frames := awaitSessionErrors(t, host, 3)
	if detail, _ := frames[0]["error"].(string); !strings.HasPrefix(detail, "claude exited") {
		t.Errorf("first error = %q, want the signaled turn's frame", detail)
	}
	for i, f := range frames[1:] {
		if f["error"] != cancelledBeforeStartDetail {
			t.Errorf("error %d = %v, want %q", i+1, f["error"], cancelledBeforeStartDetail)
		}
	}
	waitForRelease(t, client)

	if n := runCount(t, markers); n != 1 {
		t.Errorf("the fake claude ran %d turns, want only the one that was running at the cancel", n)
	}
	if _, err := os.Stat(markers.terminated); err != nil {
		t.Errorf("the running turn was not signaled: %v", err)
	}
}

// TestQueuedTurnsRunWithoutACancel pins what the drain must not break: a
// session with no cancel runs every queued turn, and each reports a result.
func TestQueuedTurnsRunWithoutACancel(t *testing.T) {
	markers := fakeClaude(t)
	client, host, conn := startCancelClient(t)

	dispatchTurn(t, conn)
	waitForFile(t, markers.started, 20*time.Second, "the fake claude never started")
	queueFollowUps(t, client, conn, 2)

	// The release file stays, so the queued turns do not wait.
	if err := os.WriteFile(markers.release, nil, 0o644); err != nil {
		t.Fatalf("release the turns: %v", err)
	}
	for i := range 3 {
		got := host.awaitFrame(t, MessageTypeResult, 20*time.Second)
		if got.frame["agent_session_id"] != cancelSession {
			t.Errorf("result %d agent_session_id = %v, want %s", i, got.frame["agent_session_id"], cancelSession)
		}
	}
	waitForRelease(t, client)

	if n := runCount(t, markers); n != 3 {
		t.Errorf("the fake claude ran %d turns, want 3", n)
	}
}

// TestCancelBeforeTheProcessStartsStopsTheTurn is the second fault of issue
// #212: a cancel between the dispatch and the process start found no process
// and did nothing, and the turn ran whole. The hook holds the turn in exactly
// that window, so no sleep has to find it.
//
// A message that arrives after the cancel is a new request. It joins the same
// worker while the first turn is held, so it also proves that the cancel
// reaches only the turn that was in flight.
func TestCancelBeforeTheProcessStartsStopsTheTurn(t *testing.T) {
	markers := fakeClaude(t)
	if err := os.WriteFile(markers.release, nil, 0o644); err != nil {
		t.Fatalf("write the release file: %v", err)
	}

	inWindow := make(chan struct{})
	proceed := make(chan struct{})
	var held atomic.Bool
	beforeTurnStart = func() {
		// Hold only the first turn. The turn after the cancel runs through.
		if held.CompareAndSwap(false, true) {
			close(inWindow)
			<-proceed
		}
	}
	t.Cleanup(func() { beforeTurnStart = func() {} })

	client, host, conn := startCancelClient(t)
	dispatchTurn(t, conn)

	select {
	case <-inWindow:
	case <-time.After(20 * time.Second):
		t.Fatal("the turn never reached the start of its process")
	}
	if removed := client.pool.Cancel(cancelSession); removed != 0 {
		t.Errorf("the cancel removed %d queued messages, want 0", removed)
	}
	queueFollowUps(t, client, conn, 1)
	close(proceed)

	frames := awaitSessionErrors(t, host, 1)
	if frames[0]["error"] != cancelledBeforeStartDetail {
		t.Errorf("error = %v, want %q", frames[0]["error"], cancelledBeforeStartDetail)
	}
	host.awaitFrame(t, MessageTypeResult, 20*time.Second)
	// The release also orders the last read of the hook before the cleanup
	// that restores it.
	waitForRelease(t, client)
	if n := runCount(t, markers); n != 1 {
		t.Errorf("the fake claude ran %d turns, want 1: only the message after the cancel", n)
	}
}
