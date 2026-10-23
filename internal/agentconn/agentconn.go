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
	// SecretKeyFingerprint is the fingerprint of the recipient key of the
	// secrets, and SecretKeyError is why the agent refused its key file
	// (specs/sandbox-secret-recipient-key/). The record never holds a key.
	SecretKeyFingerprint string `json:"secret_key_fingerprint,omitempty"`
	SecretKeyError       string `json:"secret_key_error,omitempty"`
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
	r, problem := parse(answer)
	if problem != "" {
		return problem, problem
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

// DescribeSecretKey turns the SSH answer for the record into the status value
// of the recipient key: its fingerprint, or "none" with the reason why the
// agent refused its key file.
func DescribeSecretKey(answer string) string {
	r, problem := parse(answer)
	switch {
	case problem != "":
		return problem
	case r.SecretKeyFingerprint != "":
		return r.SecretKeyFingerprint
	case r.SecretKeyError != "":
		return "none (" + r.SecretKeyError + ")"
	}
	return "unknown (no fingerprint in the record; the agent-client may be older than this check)"
}

// parse reads the SSH answer for the record. problem is the status value
// when the answer is not a record.
func parse(answer string) (r Record, problem string) {
	switch answer {
	case Unknown:
		return r, Unknown
	case Missing:
		return r, "unknown (no record; the agent-client may be older than this check)"
	}
	if err := json.Unmarshal([]byte(answer), &r); err != nil {
		return r, "unknown (the record is not valid)"
	}
	return r, ""
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
