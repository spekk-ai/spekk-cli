// Package agentconn is the record of a sandbox agent's connection to the
// control host. The agent client writes it, and `spekk sandbox status` reads
// it over SSH (specs/agent-handshake-visible/).
//
// The record exists because a refused agent looks healthy everywhere else:
// the process runs, so the systemd unit is active, and the only trace of the
// refusal is one log line for each dial.
package agentconn

import (
	"encoding/json"
	"fmt"
	"time"
)

// Path is where the agent client keeps the record. Every provision path makes
// /var/log/spekk and gives it to the agent user.
const Path = "/var/log/spekk/agent-connection.json"

// TimeLayout is RFC 3339 in UTC with milliseconds, so that a failure and a
// success in the same second still have an order. time.RFC3339 parses it.
const TimeLayout = "2006-01-02T15:04:05.000Z07:00"

// Record is what the agent last learned about its handshake. Each time is in
// TimeLayout, and an empty field means it has not happened.
type Record struct {
	LastConnectedAt string `json:"last_connected_at,omitempty"`
	LastFailureAt   string `json:"last_handshake_failure_at,omitempty"`
	// LastFailureStatus is the HTTP status of the failed handshake, or 0 when
	// the dial got no HTTP response.
	LastFailureStatus int    `json:"last_handshake_failure_status,omitempty"`
	LastFailureError  string `json:"last_handshake_failure_error,omitempty"`
}

// Missing and Unknown are the SSH answers that are not a record: the file is
// not on the machine, or SSH failed.
const (
	Missing = "missing"
	Unknown = "unknown"
)

// notConnectedWarning ends the failure line of an agent whose last dial
// failed.
const notConnectedWarning = "  WARNING: the agent is not connected to the control host"

// Describe turns the SSH answer for the record into the two status values:
// the last connection and the last handshake failure.
func Describe(answer string) (connected, failure string) {
	switch answer {
	case Unknown:
		return Unknown, Unknown
	case Missing:
		const none = "unknown (no record; the agent-client may be older than this check)"
		return none, none
	}
	var r Record
	if err := json.Unmarshal([]byte(answer), &r); err != nil {
		const bad = "unknown (the record is not valid)"
		return bad, bad
	}

	connected = orNever(r.LastConnectedAt)
	if r.LastFailureAt == "" {
		return connected, "none"
	}
	failure = r.LastFailureAt
	if r.LastFailureStatus != 0 {
		failure += fmt.Sprintf(" (HTTP %d)", r.LastFailureStatus)
	} else if r.LastFailureError != "" {
		failure += " (" + r.LastFailureError + ")"
	}
	if failedLast(r) {
		failure += notConnectedWarning
	}
	return connected, failure
}

// failedLast reports whether the last dial of the record failed. A time that
// does not parse counts as a failure that is last: an agent that cannot be
// shown to be connected must not look healthy.
func failedLast(r Record) bool {
	if r.LastConnectedAt == "" {
		return true
	}
	ok, err := time.Parse(time.RFC3339, r.LastConnectedAt)
	if err != nil {
		return true
	}
	failed, err := time.Parse(time.RFC3339, r.LastFailureAt)
	if err != nil {
		return true
	}
	return failed.After(ok)
}

func orNever(s string) string {
	if s == "" {
		return "never"
	}
	return s
}
