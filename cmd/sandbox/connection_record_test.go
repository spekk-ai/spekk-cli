package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/spekk-ai/spekk-cli/internal/agentconn"
)

// TestRecordShowsARefusedHandshakeAndTheNextSuccess is issue #215: an agent
// that the control host refused looked healthy at every surface, because the
// process ran. The server refuses the first dial with 403, the way the
// control host refuses a token that is not valid, and accepts the next.
func TestRecordShowsARefusedHandshakeAndTheNextSuccess(t *testing.T) {
	var dials atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dials.Add(1) == 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	path := filepath.Join(t.TempDir(), "agent-connection.json")
	host := "localhost:" + strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	client := NewAgentClient(Config{Token: "t", Host: host, Workspace: t.TempDir(), ConnectionRecord: path})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go client.Run(ctx)

	read := func() agentconn.Record {
		var r agentconn.Record
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &r)
		}
		return r
	}

	awaitRecord(t, "the record never showed the refused handshake", func() bool {
		return read().LastFailureStatus == http.StatusForbidden
	})
	if got := read(); got.LastConnectedAt != "" {
		t.Errorf("a refused agent records a connection at %s", got.LastConnectedAt)
	}

	// The backoff after a failure is reconnectBase, so the next dial comes
	// within seconds.
	awaitRecord(t, "the record never showed the next success", func() bool {
		return read().LastConnectedAt != ""
	})
	got := read()
	if got.LastFailureStatus != http.StatusForbidden || got.LastFailureAt == "" {
		t.Errorf("the success lost the last failure: %+v", got)
	}
	if !strings.Contains(got.LastFailureError, "403") {
		t.Errorf("failure error = %q, want the handshake error", got.LastFailureError)
	}

	// A restart starts from the record on the disk.
	if again := newConnectionRecorder(path); again.record != got {
		t.Errorf("a restart read %+v, want %+v", again.record, got)
	}
}

// awaitRecord polls cond for up to 20 seconds, and fails the test with msg.
func awaitRecord(t *testing.T, msg string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(msg)
}
