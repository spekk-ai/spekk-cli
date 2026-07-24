package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

// requestFileExt is the extension of a finalized request file. The writer
// (spekk conversation open) stages each request as "<name>.json.tmp" and
// atomically renames it to "<name>.json"; drainSpool only ever considers
// files with this extension so it can never observe — and destroy — a
// staging file whose rename has not yet committed.
const requestFileExt = ".json"

// frameSender is the narrow boundary drainSpool sends built frames through.
// In production it wraps wsjson.Write on the session's WebSocket connection;
// tests substitute a fake to observe emitted frames without a real conn.
type frameSender func(ctx context.Context, frame map[string]any) error

// provisionSpool creates a new, private spool directory for one claude
// invocation and returns it along with a cleanup func that removes it. The
// caller defers cleanup so the spool never outlives the session it belongs
// to.
func provisionSpool() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "spekk-conversation-spool-*")
	if err != nil {
		return "", nil, err
	}
	return dir, func() { os.RemoveAll(dir) }, nil
}

// drainSpool reads every request file currently sitting in spoolDir and, for
// each one, sends at most one conversation_open frame stamped with
// sessionID. Every file it considers is removed before drainSpool returns —
// whether it produced a frame, failed to send, or was dropped as invalid —
// so nothing is ever buffered for a later drain to re-consider.
func drainSpool(ctx context.Context, spoolDir, sessionID string, send frameSender) {
	entries, err := os.ReadDir(spoolDir)
	if err != nil {
		log.Printf("conversation spool: read %s: %v", spoolDir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip staging files (e.g. "request-*.json.tmp") a concurrent writer
		// has created but not yet renamed into place. Consuming one would
		// remove it out from under the writer, failing its rename and losing
		// the request.
		if !strings.HasSuffix(entry.Name(), requestFileExt) {
			continue
		}
		drainOne(ctx, filepath.Join(spoolDir, entry.Name()), sessionID, send)
	}
}

// drainOne handles a single request file: decode, validate, build the frame,
// send it, and unconditionally remove the file (fire-once).
func drainOne(ctx context.Context, path, sessionID string, send frameSender) {
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("conversation spool: read %s: %v", path, err)
		return
	}

	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("conversation spool: dropping %s: malformed request: %v", path, err)
		return
	}
	if req.Title == "" && req.Body == "" {
		log.Printf("conversation spool: dropping %s: request has neither title nor body", path)
		return
	}
	if sessionID == "" {
		log.Printf("conversation spool: dropping %s: no session id known yet", path)
		return
	}

	frame, err := NewConversationOpenFrame(sessionID, req.Title, req.Body, req.Severity, nil)
	if err != nil {
		log.Printf("conversation spool: dropping %s: %v", path, err)
		return
	}

	if err := send(ctx, frame); err != nil {
		log.Printf("conversation spool: send frame for %s: %v", path, err)
	}
}
