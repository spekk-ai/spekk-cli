// Package conversation is the single source of truth for the
// conversation-open request-file contract shared between the process that
// writes a request (a CLI subcommand run as an agent tool) and the process
// that drains those requests and forwards them onward (a long-running
// worker). It declares only the contract itself — the spool
// environment-variable name, the request-file shape, and the set of allowed
// severities — so both sides can depend on it without pulling in each
// other's concerns.
package conversation

// SpoolEnvVar is the name of the environment variable that points a session
// at its private spool directory: the writer drops request files there, and
// the drainer reads them from there. Both sides must agree on this exact
// variable name, which is why it lives here instead of being declared
// independently in each.
const SpoolEnvVar = "SPEKK_CONVERSATION_SPOOL"

// Severity is the urgency of a conversation-open request.
type Severity string

// The complete, closed set of valid severities. Any value outside this set
// is invalid — see IsValidSeverity.
const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// DefaultSeverity is used when a request does not specify a severity.
const DefaultSeverity = SeverityInfo

// Request-file naming is part of the contract: the writer stages a request
// as "<name>.json.tmp" and atomically renames it to "<name>.json"; the
// drainer only ever considers finalized ".json" files, so it can never
// observe — and destroy — a staging file whose rename has not yet
// committed. Both sides must agree on these exact suffixes: if they drift,
// finalized requests are silently ignored.
const (
	// RequestFileExt is the extension of a finalized request file — the only
	// kind the drainer consumes.
	RequestFileExt = ".json"
	// StagingSuffix is appended to RequestFileExt while the writer is still
	// writing; the atomic rename strips it.
	StagingSuffix = ".tmp"
	// RequestFilePattern is the os.CreateTemp pattern the writer stages new
	// request files with.
	RequestFilePattern = "request-*" + RequestFileExt + StagingSuffix
)

// Request is the on-disk shape of one conversation-open request file. It
// carries only what the requester supplies; the session id is not part of
// this contract because the request file never carries it — the drainer
// stamps the authoritative session id itself when it forwards the request.
type Request struct {
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Severity Severity `json:"severity"`
}

// IsValidSeverity reports whether s is one of the three known severities.
// Anything else — including an empty string — is invalid.
func IsValidSeverity(s string) bool {
	switch Severity(s) {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}
