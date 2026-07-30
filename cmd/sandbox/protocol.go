package main

import (
	"strings"

	"github.com/coder/websocket"
)

// ProtocolVersion is the WebSocket contract version this client speaks
// (specs/sandbox-protocol-version/). It is sent as the X-Spekk-Protocol
// header on every dial, and compared against the server's welcome frame.
//
// Bump rules: a breaking change to message types, frame fields, or close
// codes bumps the major. An additive change bumps the minor. A PR that
// bumps the major names the companion spekk-app PR (server spec:
// protocol-handshake).
const ProtocolVersion = "1.0"

// protocolHeaderName carries the version on dial.
const protocolHeaderName = "X-Spekk-Protocol"

// protocolRejectedCloseCode is the close code the control host sends when
// it refuses this client's major version.
const protocolRejectedCloseCode = websocket.StatusCode(4004)

// protocolMajor returns the text before the first dot. A value with no dot
// is returned whole — the comparison then simply mismatches instead of
// panicking.
func protocolMajor(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}

// isProtocolReject reports whether a connection error is the control
// host's protocol rejection.
func isProtocolReject(err error) bool {
	return websocket.CloseStatus(err) == protocolRejectedCloseCode
}
