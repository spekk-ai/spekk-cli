package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeRequestFile writes raw content into dir/name, failing the test on error.
func writeRequestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write request file %s: %v", path, err)
	}
	return path
}

// --- drainSpool: fire-once ---

// TestDrainSpoolFireOnce verifies a valid request produces exactly one
// frame stamped with the worker's session id, and that the request file is
// removed whether the send succeeds or fails.
func TestDrainSpoolFireOnce(t *testing.T) {
	t.Run("send succeeds", func(t *testing.T) {
		dir := t.TempDir()
		path := writeRequestFile(t, dir, "req.json", `{"title":"T","body":"B","severity":"warning"}`)

		var got map[string]any
		send := func(ctx context.Context, frame map[string]any) error {
			got = frame
			return nil
		}

		drainSpool(context.Background(), dir, "sess-abc", send)

		if got == nil {
			t.Fatal("expected a frame to be sent")
		}
		if got["session_id"] != "sess-abc" {
			t.Errorf("session_id = %v, want sess-abc (the worker's id, not from the file)", got["session_id"])
		}
		if got["title"] != "T" || got["body"] != "B" {
			t.Errorf("unexpected frame contents: %v", got)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected request file removed after send, stat err=%v", err)
		}
	})

	t.Run("send fails", func(t *testing.T) {
		dir := t.TempDir()
		path := writeRequestFile(t, dir, "req.json", `{"title":"T","body":"B","severity":"info"}`)

		send := func(ctx context.Context, frame map[string]any) error {
			return fmt.Errorf("boom")
		}

		drainSpool(context.Background(), dir, "sess-abc", send)

		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected request file removed even though send failed, stat err=%v", err)
		}
	})
}

// --- drainSpool: session id not yet known ---

// TestDrainSpoolDropsRequestBeforeSessionIDKnown verifies that a request
// drained before any session id is known is dropped: no frame is sent, and
// the file is still removed rather than left for a later drain to retry
// (no buffering).
func TestDrainSpoolDropsRequestBeforeSessionIDKnown(t *testing.T) {
	dir := t.TempDir()
	path := writeRequestFile(t, dir, "req.json", `{"title":"T","body":"B","severity":"info"}`)

	sent := false
	send := func(ctx context.Context, frame map[string]any) error {
		sent = true
		return nil
	}

	drainSpool(context.Background(), dir, "", send)

	if sent {
		t.Error("expected no frame sent when session id is not yet known")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected request file removed (not buffered), stat err=%v", err)
	}
}

// --- drainSpool: malformed requests ---

// TestDrainSpoolDropsMalformedRequests covers the three ways a request file
// can be invalid — bad JSON, an out-of-range severity, and neither title nor
// body set — verifying each is dropped (no frame, no panic) and removed.
func TestDrainSpoolDropsMalformedRequests(t *testing.T) {
	cases := map[string]string{
		"bad json":             `{not valid json`,
		"invalid severity":     `{"title":"T","body":"B","severity":"urgent"}`,
		"empty title and body": `{"title":"","body":"","severity":"info"}`,
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := writeRequestFile(t, dir, "req.json", content)

			sent := false
			send := func(ctx context.Context, frame map[string]any) error {
				sent = true
				return nil
			}

			drainSpool(context.Background(), dir, "sess-1", send)

			if sent {
				t.Errorf("expected no frame sent for malformed request (%s)", name)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("expected malformed request file removed, stat err=%v", err)
			}
		})
	}
}

// TestDrainSpoolContinuesAfterMalformedFile verifies one malformed file
// among several does not stop the scan: the valid sibling still produces a
// frame, and both files end up removed.
func TestDrainSpoolContinuesAfterMalformedFile(t *testing.T) {
	dir := t.TempDir()
	writeRequestFile(t, dir, "a-bad.json", `not json`)
	writeRequestFile(t, dir, "b-good.json", `{"title":"T","body":"B","severity":"info"}`)

	var frames []map[string]any
	send := func(ctx context.Context, frame map[string]any) error {
		frames = append(frames, frame)
		return nil
	}

	drainSpool(context.Background(), dir, "sess-1", send)

	if len(frames) != 1 {
		t.Fatalf("expected exactly 1 frame, got %d: %v", len(frames), frames)
	}
	if frames[0]["title"] != "T" {
		t.Errorf("unexpected frame: %v", frames[0])
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read spool dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected spool dir empty after drain, got %d entries", len(entries))
	}
}

// --- provisionSpool ---

// TestProvisionSpoolCreatesAndCleansUpDir verifies the spool directory
// exists once provisioned and is gone once cleanup runs (as invoke() defers
// it at session end).
func TestProvisionSpoolCreatesAndCleansUpDir(t *testing.T) {
	dir, cleanup, err := provisionSpool()
	if err != nil {
		t.Fatalf("provisionSpool: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected spool dir to exist after provisioning, stat err=%v", err)
	}

	cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected spool dir removed after cleanup, stat err=%v", err)
	}
}
