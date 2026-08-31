package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

// TestProtocolVersionPinned makes a version change a deliberate diff.
func TestProtocolVersionPinned(t *testing.T) {
	if ProtocolVersion != "1.0" {
		t.Fatalf("ProtocolVersion changed to %q — bump deliberately and update the companion control-host PR", ProtocolVersion)
	}
}

func TestProtocolMajor(t *testing.T) {
	cases := map[string]string{"1.0": "1", "2.3": "2", "weird": "weird", "": ""}
	for in, want := range cases {
		if got := protocolMajor(in); got != want {
			t.Fatalf("protocolMajor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDialOptionsSendBothHeaders(t *testing.T) {
	c := &AgentClient{cfg: Config{Token: "tok"}}
	h := c.dialOptions().HTTPHeader
	if h.Get("Authorization") != "Bearer tok" {
		t.Fatalf("Authorization header missing or wrong: %q", h.Get("Authorization"))
	}
	if h.Get(protocolHeaderName) != ProtocolVersion {
		t.Fatalf("%s header missing or wrong: %q", protocolHeaderName, h.Get(protocolHeaderName))
	}
}

func TestWelcomeFrameIsNotUnknown(t *testing.T) {
	c := &AgentClient{}
	buf := captureLog(t)
	c.handleInbound(t.Context(), Message{Type: MessageTypeWelcome, Protocol: ProtocolVersion})
	out := buf.String()
	if strings.Contains(out, "Unknown message type") {
		t.Fatalf("welcome treated as unknown:\n%s", out)
	}
	if strings.Contains(out, "WARNING") {
		t.Fatalf("same major must not warn:\n%s", out)
	}
}

func TestWelcomeMajorMismatchWarns(t *testing.T) {
	c := &AgentClient{}
	buf := captureLog(t)
	c.handleWelcome(Message{Type: MessageTypeWelcome, Protocol: "2.0"})
	out := buf.String()
	if !strings.Contains(out, "WARNING") || !strings.Contains(out, "2.0") || !strings.Contains(out, ProtocolVersion) {
		t.Fatalf("mismatch warning must name both versions:\n%s", out)
	}
}

func TestIsProtocolReject(t *testing.T) {
	reject := websocket.CloseError{Code: protocolRejectedCloseCode}
	if !isProtocolReject(reject) {
		t.Fatal("4004 close must classify as protocol reject")
	}
	if isProtocolReject(errors.New("dial: connection refused")) {
		t.Fatal("ordinary errors must not classify as protocol reject")
	}
}
