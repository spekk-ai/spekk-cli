package agentconn

import "testing"

// TestDescribe pins what `spekk sandbox status` says about the handshake. The
// point of the record is the warning: a refused agent looked healthy at every
// surface, because its process ran (issue #215).
func TestDescribe(t *testing.T) {
	const warn = notConnectedWarning
	tests := []struct {
		name          string
		answer        string
		wantConnected string
		wantFailure   string
	}{
		{
			"a healthy agent that never failed",
			`{"last_connected_at":"2026-10-23T05:00:00.000Z"}`,
			"2026-10-23T05:00:00.000Z", "none",
		},
		{
			"an agent that never connected and is refused",
			`{"last_handshake_failure_at":"2026-10-23T05:00:00.000Z","last_handshake_failure_status":403,"last_handshake_failure_error":"failed to WebSocket dial: expected handshake response status code 101 but got 403"}`,
			"never", "2026-10-23T05:00:00.000Z (HTTP 403)" + warn,
		},
		{
			"an agent that connected and then failed with no HTTP answer",
			`{"last_connected_at":"2026-10-23T05:00:00.000Z","last_handshake_failure_at":"2026-10-23T05:00:00.500Z","last_handshake_failure_error":"dial tcp: connection refused"}`,
			"2026-10-23T05:00:00.000Z", "2026-10-23T05:00:00.500Z (dial tcp: connection refused)" + warn,
		},
		{
			"an agent that failed and then connected again",
			`{"last_connected_at":"2026-10-23T05:01:00.000Z","last_handshake_failure_at":"2026-10-23T05:00:00.000Z","last_handshake_failure_status":502}`,
			"2026-10-23T05:01:00.000Z", "2026-10-23T05:00:00.000Z (HTTP 502)",
		},
		{
			"no record on the machine",
			Missing,
			"unknown (no record; the agent-client may be older than this check)",
			"unknown (no record; the agent-client may be older than this check)",
		},
		{"SSH failed", Unknown, Unknown, Unknown},
		{"a record that is not valid", `{"last_connected_at":`, "unknown (the record is not valid)", "unknown (the record is not valid)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connected, failure := Describe(tt.answer)
			if connected != tt.wantConnected {
				t.Errorf("connected = %q, want %q", connected, tt.wantConnected)
			}
			if failure != tt.wantFailure {
				t.Errorf("failure = %q, want %q", failure, tt.wantFailure)
			}
		})
	}
}

// TestDescribeSecretKey pins the fingerprint line of status. An admin
// compares it with the Secrets section of spekk-app before accepting a key,
// so a refused key or an old agent must never look like a fingerprint.
func TestDescribeSecretKey(t *testing.T) {
	tests := []struct{ name, answer, want string }{
		{"a usable key", `{"last_connected_at":"2026-10-23T05:00:00.000Z","secret_key_fingerprint":"300c9c9603b92a4b"}`, "300c9c9603b92a4b"},
		{"a refused key file", `{"secret_key_error":"/home/agent/.config/spekk-agent/secret-key has mode 0644"}`, "none (/home/agent/.config/spekk-agent/secret-key has mode 0644)"},
		{"an agent older than the key", `{"last_connected_at":"2026-10-23T05:00:00.000Z"}`, "unknown (no fingerprint in the record; the agent-client may be older than this check)"},
		{"no record on the machine", Missing, "unknown (no record; the agent-client may be older than this check)"},
		{"SSH failed", Unknown, Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DescribeSecretKey(tt.answer); got != tt.want {
				t.Errorf("DescribeSecretKey = %q, want %q", got, tt.want)
			}
		})
	}
}
