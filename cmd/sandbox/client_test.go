package main

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// captureLog redirects the standard logger's output to a buffer for the
// duration of the test and restores it on cleanup.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })
	return &buf
}

// --- inbound error frames ---

// TestHandleInboundErrorFrameConversationOpenCode verifies each documented
// conversation_open rejection code is logged with both its code and detail,
// and that routing through handleInbound (the inbound-handling path) does
// not fall through to the "Unknown message type" branch.
func TestHandleInboundErrorFrameConversationOpenCode(t *testing.T) {
	codes := []string{
		"conversation_open_invalid",
		"conversation_open_no_channel",
		"conversation_open_failed",
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			buf := captureLog(t)
			c := &AgentClient{}

			msg := Message{Type: MessageTypeError, Error: code, Detail: "no channel bound to session"}
			c.handleInbound(context.Background(), nil, msg)

			got := buf.String()
			if !strings.Contains(got, code) {
				t.Errorf("log output %q does not contain error code %q", got, code)
			}
			if !strings.Contains(got, "no channel bound to session") {
				t.Errorf("log output %q does not contain detail", got)
			}
			if strings.Contains(got, "Unknown message type") {
				t.Errorf("error frame was treated as unknown message type: %q", got)
			}
		})
	}
}

// TestHandleInboundErrorFrameUnknownCode verifies an error frame with a code
// the worker doesn't specifically recognize is still logged (code + detail)
// rather than silently swallowed or crashing.
func TestHandleInboundErrorFrameUnknownCode(t *testing.T) {
	buf := captureLog(t)
	c := &AgentClient{}

	msg := Message{Type: MessageTypeError, Error: "some_future_code", Detail: "something else went wrong"}
	c.handleInbound(context.Background(), nil, msg)

	got := buf.String()
	if !strings.Contains(got, "some_future_code") {
		t.Errorf("log output %q does not contain error code", got)
	}
	if !strings.Contains(got, "something else went wrong") {
		t.Errorf("log output %q does not contain detail", got)
	}
}

// TestHandleInboundNonErrorFramesUnaffected verifies frame types unrelated
// to this change (heartbeat_ack and an unknown type) still behave as
// before: heartbeat_ack is silently ignored, and a truly unknown type still
// hits the "Unknown message type" branch.
func TestHandleInboundNonErrorFramesUnaffected(t *testing.T) {
	t.Run("heartbeat_ack is ignored", func(t *testing.T) {
		buf := captureLog(t)
		c := &AgentClient{}

		c.handleInbound(context.Background(), nil, Message{Type: MessageTypeHeartbeatAck})

		if got := buf.String(); got != "" {
			t.Errorf("expected no log output for heartbeat_ack, got %q", got)
		}
	})

	t.Run("unknown type still logged as unknown", func(t *testing.T) {
		buf := captureLog(t)
		c := &AgentClient{}

		c.handleInbound(context.Background(), nil, Message{Type: "something_new"})

		got := buf.String()
		if !strings.Contains(got, "Unknown message type: something_new") {
			t.Errorf("expected unknown message type log, got %q", got)
		}
	})
}
