package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// The kinds of sandbox. A one-off sandbox is for one task and then a
// destroy. A persistent sandbox runs a long-lived agent.
const (
	KindOneOff     = "one-off"
	KindPersistent = "persistent"
)

// defaultOneOffExpiry is how long a one-off sandbox lives when --expires
// does not say. Expiry is a reminder in list, never a destroy.
const defaultOneOffExpiry = 48 * time.Hour

// sandboxMarker is the file on the machine that says what kind of sandbox
// it is, for whoever logs in. Credential injection writes it, and the
// teardown of a machine you already have removes it. It is a variable so
// that a test of the teardown does not remove the marker of the machine
// that runs it.
var sandboxMarker = "/opt/spekk/sandbox.json"

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

// ResolveKind turns the create flags into a kind and an expiry. A one-off
// sandbox expires defaultOneOffExpiry after now unless expires says when.
// A persistent sandbox has no expiry, so --expires without --one-off is an
// error rather than a sandbox with two meanings.
func ResolveKind(oneOff bool, expires string, now time.Time) (string, time.Time, error) {
	if !oneOff {
		if expires != "" {
			return "", time.Time{}, fmt.Errorf("--expires needs --one-off: a persistent sandbox does not expire")
		}
		return KindPersistent, time.Time{}, nil
	}
	if expires == "" {
		return KindOneOff, now.Add(defaultOneOffExpiry).UTC(), nil
	}
	t, err := parseExpiry(expires, now)
	if err != nil {
		return "", time.Time{}, err
	}
	return KindOneOff, t, nil
}

// ValidatePurpose refuses a --purpose with a control character (C0, DEL or
// C1). status prints the purpose as one "Key: value" line, so a newline
// breaks the layout and an escape sequence goes to the terminal. Every other
// character is kept as it is.
func ValidatePurpose(purpose string) error {
	for _, r := range purpose {
		if unicode.IsControl(r) {
			return fmt.Errorf("--purpose has a control character (%U): use one line of plain text", r)
		}
	}
	return nil
}

// maxExpiryDays is the largest day count that fits in a time.Duration.
const maxExpiryDays = int(math.MaxInt64 / int64(24*time.Hour))

// parseExpiry reads an --expires value: a Go duration (36h), a number of
// days (3d), a date (2026-10-17, which means 00:00 UTC on that day), or an
// RFC 3339 time. The result is in UTC and must be after now.
func parseExpiry(s string, now time.Time) (time.Time, error) {
	var t time.Time
	if d, err := time.ParseDuration(s); err == nil {
		t = now.Add(d)
	} else if n, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil && strings.HasSuffix(s, "d") {
		if n > maxExpiryDays || n < -maxExpiryDays {
			return time.Time{}, fmt.Errorf("--expires %q is too far away: use %d days or fewer", s, maxExpiryDays)
		}
		t = now.Add(time.Duration(n) * 24 * time.Hour)
	} else if d, err := time.Parse(time.DateOnly, s); err == nil {
		t = d
	} else if d, err := time.Parse(time.RFC3339, s); err == nil {
		t = d
	} else {
		return time.Time{}, fmt.Errorf("invalid --expires %q: use a duration (36h), days (3d), a date (2026-10-17) or an RFC 3339 time", s)
	}
	if !t.After(now) {
		return time.Time{}, fmt.Errorf("--expires %q is not in the future", s)
	}
	return t.UTC(), nil
}

// Labeler is a Provider that can put labels on its machines. It is optional:
// Create finds it with a type assertion, and a provider that cannot label
// does not implement it. Labels let a machine that did not create a sandbox
// see its kind in the cloud.
type Labeler interface {
	// Label puts labels on the machine meta names. Each key and value is
	// already lowercase [a-z0-9_-] and 63 characters or fewer, so the same
	// label means the same thing on every provider. An error is a warning
	// to the caller, never a failed create.
	Label(meta *SandboxMeta, labels map[string]string) error

	// ListLabeled returns the machines of this provider that carry spekk
	// labels, whoever created them. Each one has the provider's own
	// identifiers, so list can match it to a local record, and the kind,
	// expiry and host that its labels say.
	ListLabeled() ([]RemoteSandbox, error)
}

// RemoteSandbox is a machine that a provider reports from its labels.
type RemoteSandbox struct {
	Name string
	Meta SandboxMeta
}

// metaFromLabels fills the kind fields of meta from a machine's labels.
// The expiry label is a date, which reads as 00:00 UTC on that day.
func metaFromLabels(meta *SandboxMeta, labels map[string]string) {
	meta.Kind = labels[labelKind]
	meta.CreatedBy = labels[labelHost]
	if t, err := time.Parse(time.DateOnly, labels[labelExpires]); err == nil {
		meta.ExpiresAt = t.Format(time.RFC3339)
	}
}

// sameMachine reports whether a local record and a remote machine are the
// same machine: the same provider and the same provider identifier.
func sameMachine(local, remote *SandboxMeta) bool {
	if providerName(local) != providerName(remote) {
		return false
	}
	switch {
	case remote.DropletID != 0:
		return local.DropletID == remote.DropletID
	case remote.GCPInstance != "":
		return local.GCPInstance == remote.GCPInstance && local.Project == remote.Project && local.Region == remote.Region
	}
	return false
}

// providerByName is the seam list --all-hosts builds providers through, so
// a test can use fake providers and no cloud API.
var providerByName = ProviderByName

// remoteRows asks each provider that labels for its machines, and returns
// those that no local record names. For each cloud provider it cannot ask,
// it writes one line to errOut, so no cloud is left out silently. A provider
// that does not label (none) is not a cloud, so it gets no line.
func remoteRows(local map[string]*SandboxMeta, errOut io.Writer) []listRow {
	var rows []listRow
	for _, name := range ValidProviders {
		p, err := providerByName(name)
		if err != nil {
			fmt.Fprintf(errOut, "%s: cannot query: %s\n", name, err)
			continue
		}
		l, ok := p.(Labeler)
		if !ok {
			continue
		}
		found, err := l.ListLabeled()
		if err != nil {
			fmt.Fprintf(errOut, "%s: cannot query: %s\n", name, err)
			continue
		}
	machines:
		for _, r := range found {
			for _, meta := range local {
				if sameMachine(meta, &r.Meta) {
					continue machines
				}
			}
			meta := r.Meta
			meta.Status = "remote"
			rows = append(rows, listRow{r.Name, &meta})
		}
	}
	return rows
}

// The label keys spekk puts on a machine.
const (
	labelKind    = "spekk-kind"
	labelExpires = "spekk-expires"
	labelHost    = "spekk-host"
)

// sandboxLabels returns the labels for a sandbox. The expiry label is the
// date only, rounded up to the next day, so a list that reads the label
// flags a sandbox late rather than early.
func sandboxLabels(meta *SandboxMeta) map[string]string {
	labels := map[string]string{labelKind: labelValue(meta.KindOf())}
	if meta.CreatedBy != "" {
		labels[labelHost] = labelValue(meta.CreatedBy)
	}
	if t, err := time.Parse(time.RFC3339, meta.ExpiresAt); err == nil {
		t = t.UTC()
		day := t.Truncate(24 * time.Hour)
		if day.Before(t) {
			day = day.Add(24 * time.Hour)
		}
		labels[labelExpires] = day.Format(time.DateOnly)
	}
	return labels
}

// labelValue makes s a valid label value under the strictest rules, which
// are GCP's: lowercase [a-z0-9_-], 63 characters or fewer. Any other
// character becomes "-".
func labelValue(s string) string {
	b := []byte(strings.ToLower(s))
	for i, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			b[i] = '-'
		}
	}
	if len(b) > 63 {
		b = b[:63]
	}
	return string(b)
}

// sortedLabels returns the labels as key<sep>value strings, sorted by key,
// so a provider sends them in a stable order.
func sortedLabels(labels map[string]string, sep string) []string {
	out := make([]string, 0, len(labels))
	for k, v := range labels {
		out = append(out, k+sep+v)
	}
	sort.Strings(out)
	return out
}

// labelMachine labels the machine when the provider can, with the kind
// labels and the extra labels of the caller. A failure is a warning: the
// record and the marker still carry the kind.
func labelMachine(p Provider, meta *SandboxMeta, extra map[string]string) {
	l, ok := p.(Labeler)
	if !ok {
		return
	}
	labels := sandboxLabels(meta)
	for k, v := range extra {
		labels[k] = v
	}
	if err := l.Label(meta, labels); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not label %s: %s\n", machineRef(meta), err)
	}
}

// markerContent is the JSON that credential injection writes to
// sandboxMarker. It holds no secret.
func markerContent(meta *SandboxMeta) []byte {
	data, _ := json.MarshalIndent(struct {
		Kind      string `json:"kind"`
		Purpose   string `json:"purpose"`
		ExpiresAt string `json:"expiresAt"`
		CreatedBy string `json:"createdBy"`
	}{meta.KindOf(), meta.Purpose, meta.ExpiresAt, meta.CreatedBy}, "", "  ")
	return append(data, '\n')
}

// hostname is the name recorded as createdBy. A seam for tests.
var hostname = func() string {
	h, err := os.Hostname()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read the hostname, so the record has no createdBy: %s\n", err)
	}
	return h
}
