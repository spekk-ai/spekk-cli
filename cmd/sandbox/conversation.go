package main

import (
	"fmt"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

// NewConversationOpenFrame builds the single outbound conversation_open
// frame the worker sends to the control host over the WebSocket (the same
// way the other outbound frames are written, e.g. via wsjson.Write). It is
// the one place that knows the frame's shape and encoding rules:
//
//   - session_id must be non-empty; conversation_open is meaningless without
//     it, so an empty session_id is rejected rather than sent.
//   - severity must be one of the values from the shared conversation
//     package. An empty severity defaults to conversation.DefaultSeverity;
//     any other invalid value is rejected rather than passed through.
//   - metadata is optional. When empty, the key is omitted from the frame
//     entirely rather than serialized as null.
//
// conversation_open is worker→control-host only: it is not part of the
// inbound Message struct and is never read in readLoop.
func NewConversationOpenFrame(sessionID, title, body string, severity conversation.Severity, metadata map[string]any) (map[string]any, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("conversation_open: session_id must not be empty")
	}

	sev := severity
	switch {
	case sev == "":
		sev = conversation.DefaultSeverity
	case !conversation.IsValidSeverity(string(sev)):
		return nil, fmt.Errorf("conversation_open: invalid severity %q", severity)
	}

	frame := map[string]any{
		"type":       MessageTypeConversationOpen,
		"session_id": sessionID,
		"title":      title,
		"body":       body,
		"severity":   sev,
	}
	if len(metadata) > 0 {
		frame["metadata"] = metadata
	}
	return frame, nil
}
