package sandbox

import "time"

// The kinds of sandbox. A one-off sandbox is for one task and then a
// destroy. A persistent sandbox runs a long-lived agent.
const (
	KindOneOff     = "one-off"
	KindPersistent = "persistent"
)

// KindOf returns the kind of a sandbox. A record with no kind was written
// before the field existed, when every sandbox was persistent.
func (m *SandboxMeta) KindOf() string {
	if m.Kind == "" {
		return KindPersistent
	}
	return m.Kind
}

// Expired reports whether the sandbox has an expiry and it is before now.
// An expiry that does not parse is not expired: it is a reminder only, and
// a false alarm is worse than no alarm.
func (m *SandboxMeta) Expired(now time.Time) bool {
	if m.ExpiresAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, m.ExpiresAt)
	return err == nil && t.Before(now)
}
