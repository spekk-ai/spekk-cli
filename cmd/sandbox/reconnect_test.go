package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// A dropped connection used to kill the turn it was carrying: the worker held
// the per-connection context, and the goroutine in invoke signaled the claude
// process when that context ended. The client then reconnected onto work that
// was already dead, and the control host saw a turn that reported nothing at
// all. These tests drive the real dispatch path across a real connection loss.

// recvFrame is a frame the test control host received, tagged with the
// connection it arrived on. The connection number is the point of the test:
// the result has to land on the connection that is live when the turn ends,
// not on the one the turn started on.
type recvFrame struct {
	conn  int
	frame map[string]any
}

// testControlHost is a WebSocket server that accepts repeated connections from
// one client and records every frame with the connection it came in on.
type testControlHost struct {
	srv      *httptest.Server
	accepted chan *websocket.Conn
	frames   chan recvFrame

	mu    sync.Mutex
	count int
}

func newTestControlHost(t *testing.T) *testControlHost {
	t.Helper()
	h := &testControlHost{
		accepted: make(chan *websocket.Conn, 4),
		frames:   make(chan recvFrame, 64),
	}
	h.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		h.mu.Lock()
		h.count++
		n := h.count
		h.mu.Unlock()

		h.accepted <- conn
		for {
			var frame map[string]any
			if err := wsjson.Read(r.Context(), conn, &frame); err != nil {
				return
			}
			h.frames <- recvFrame{conn: n, frame: frame}
		}
	}))
	t.Cleanup(h.srv.Close)
	return h
}

// host returns the host:port the client should dial, named "localhost" rather
// than by its loopback address: wsURL only chooses ws:// over wss:// when the
// host says localhost, and the test server speaks plain ws.
func (h *testControlHost) host() string {
	return "localhost:" + strings.TrimPrefix(h.srv.URL, "http://127.0.0.1:")
}

func (h *testControlHost) nextConn(t *testing.T) *websocket.Conn {
	t.Helper()
	select {
	case conn := <-h.accepted:
		return conn
	case <-time.After(20 * time.Second):
		t.Fatal("the client never connected")
		return nil
	}
}

// awaitFrame waits for the next frame of the given type and returns it with
// the connection number it arrived on. Frames of other types are skipped, so
// anything ahead of the result in the stream does not hide the result.
func (h *testControlHost) awaitFrame(t *testing.T, frameType string, within time.Duration) recvFrame {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case got := <-h.frames:
			if got.frame["type"] == frameType {
				return got
			}
		case <-deadline:
			t.Fatalf("no %q frame arrived within %s", frameType, within)
			return recvFrame{}
		}
	}
}

// turnMarkers are the files the fake claude touches to report what happened to
// it. They are how a test tells a turn that survived from a turn that was
// signaled, without inspecting the process table.
type turnMarkers struct {
	started    string // the turn is running; safe to drop the connection
	release    string // the test creates this to let the turn finish
	finished   string // the turn ran to the end
	terminated string // the turn received SIGTERM
}

// fakeClaude puts an executable named claude on PATH for this test. The script
// reads and discards stdin the way the real binary is driven, waits for the
// test to create the release file, then emits one non-result stream line and
// one result line. Waiting on a file rather than a fixed sleep holds the turn
// open for exactly as long as the test needs the connection to drop
// underneath it.
func fakeClaude(t *testing.T) turnMarkers {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	m := turnMarkers{
		started:    filepath.Join(dir, "started"),
		release:    filepath.Join(dir, "release"),
		finished:   filepath.Join(dir, "finished"),
		terminated: filepath.Join(dir, "terminated"),
	}

	// The trap runs between sleeps, so the loop below is also the signal
	// window. cat drains the message invoke writes to stdin.
	script := fmt.Sprintf(`#!/bin/sh
trap 'touch %q; exit 143' TERM
cat > /dev/null
touch %q
while [ ! -f %q ]; do sleep 0.05; done
touch %q
echo '{"type":"system","subtype":"init","session_id":"sess-1"}'
echo '{"type":"result","session_id":"sess-1","result":"the turn finished"}'
`, m.terminated, m.started, m.release, m.finished)

	path := filepath.Join(dir, "claude")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return m
}

// dispatchTurn sends one message frame, the way the control host dispatches a
// scheduled run.
func dispatchTurn(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	err := wsjson.Write(context.Background(), conn, map[string]any{
		"type":             MessageTypeMessage,
		"text":             "do the long thing",
		"agent_session_id": "agent-session-1",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
}

// TestTurnSurvivesConnectionLossAndReportsOnNewConnection is the regression
// test for the production failure: a turn running when the WebSocket dropped
// was killed, and the control host got no reply of any kind.
//
// It covers the stream-frame case in the same run. The turn's non-result line
// is emitted while no connection is live, so that frame is dropped — and the
// turn still reaches its result rather than blocking on a send that cannot
// complete.
func TestTurnSurvivesConnectionLossAndReportsOnNewConnection(t *testing.T) {
	markers := fakeClaude(t)
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	first := host.nextConn(t)
	dispatchTurn(t, first)

	// Drop only once the turn is genuinely running. Dropping earlier would
	// test nothing.
	waitForFile(t, markers.started, 20*time.Second, "the fake claude never started")

	// Drop the connection the way the network does: no close handshake.
	first.CloseNow()

	// The client reconnects on its own, and the turn is still running.
	host.nextConn(t)

	// Let the turn finish, now that it is running with no connection at all.
	if err := os.WriteFile(markers.release, nil, 0o644); err != nil {
		t.Fatalf("release the turn: %v", err)
	}

	got := host.awaitFrame(t, MessageTypeResult, 20*time.Second)

	if got.conn != 2 {
		t.Errorf("result arrived on connection %d, want the reconnected one (2)", got.conn)
	}
	if got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("result agent_session_id = %v, want agent-session-1", got.frame["agent_session_id"])
	}
	if got.frame["output"] != "the turn finished" {
		t.Errorf("result output = %v, want the turn's real output", got.frame["output"])
	}
	if _, err := os.Stat(markers.terminated); err == nil {
		t.Error("the turn was signaled by the dropped connection")
	}
	if _, err := os.Stat(markers.finished); err != nil {
		t.Errorf("the turn did not run to the end: %v", err)
	}
}

// TestProcessShutdownStillSignalsTheTurn pins the behavior the original
// context was reaching for. A turn must still be signaled when the process
// itself is going away — that is what the goroutine in invoke is for, and
// scoping it to the process must not have made it inert.
func TestProcessShutdownStillSignalsTheTurn(t *testing.T) {
	markers := fakeClaude(t)
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		client.Run(ctx)
		close(done)
	}()

	dispatchTurn(t, host.nextConn(t))
	waitForFile(t, markers.started, 20*time.Second, "the fake claude never started")

	// The release file is never created, so this turn only ends by signal.
	cancel()

	waitForFile(t, markers.terminated, 20*time.Second,
		"the turn was not signaled when the process context ended")

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}

// TestReconnectDelayResetsAfterAGoodConnection covers the backoff ratchet.
// The delay only ever grew, so a process that had dropped a handful of times
// waited the full reconnectMax for every later reconnect, even when the
// connection before it had been healthy for hours. Every reconnect gap
// observed in production was exactly reconnectMax and never reconnectBase.
func TestReconnectDelayResetsAfterAGoodConnection(t *testing.T) {
	tests := []struct {
		name        string
		last        time.Duration
		established bool
		want        time.Duration
	}{
		{"first attempt starts at the base", 0, false, reconnectBase},
		{"consecutive failures double", reconnectBase, false, reconnectBase * 2},
		{"doubling stops at the max", reconnectMax, false, reconnectMax},
		{"a connection that worked resets the backoff", reconnectMax, true, reconnectBase},
		{"a short good connection also resets", reconnectBase * 4, true, reconnectBase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reconnectDelay(tt.last, tt.established); got != tt.want {
				t.Errorf("reconnectDelay(%s, %v) = %s, want %s", tt.last, tt.established, got, tt.want)
			}
		})
	}
}

// TestSendFinalWaitsAndSendDrops covers the two send behaviors in isolation,
// where the end-to-end test cannot separate them: a frame that ends a turn
// waits for a connection, and a stream frame does not.
func TestSendFinalWaitsAndSendDrops(t *testing.T) {
	host := newTestControlHost(t)
	holder := newConnHolder()

	sent := make(chan error, 1)
	go func() {
		sent <- holder.sendFinal(context.Background(), map[string]any{"type": MessageTypeResult})
	}()

	// Nothing is live yet, so the send waits rather than failing.
	select {
	case err := <-sent:
		t.Fatalf("sendFinal returned %v before any connection existed", err)
	case <-time.After(200 * time.Millisecond):
	}

	conn, _, err := websocket.Dial(context.Background(), "ws://"+host.host(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()
	holder.set(conn)

	select {
	case err := <-sent:
		if err != nil {
			t.Errorf("sendFinal after a connection arrived = %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("sendFinal did not deliver once a connection was published")
	}

	holder.clear()
	if err := holder.send(context.Background(), map[string]any{"type": MessageTypeStream}); !errors.Is(err, errNoConnection) {
		t.Errorf("send with no connection = %v, want errNoConnection", err)
	}
}

func waitForFile(t *testing.T, path string, within time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(msg)
}
