package sandbox

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// useLegacyFixture copies the v1.32.0 registry fixture into a temp store.
func useLegacyFixture(t *testing.T) string {
	t.Helper()
	path := useTempStore(t)
	data, err := os.ReadFile("testdata/sandboxes-v1.32.0.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readRecords decodes the store as raw JSON, so a test sees the keys on disk.
func readRecords(t *testing.T, path string) map[string]map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records map[string]map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatal(err)
	}
	return records
}

// A registry that v1.32.0 wrote lists every record as persistent, with no
// expiry, and a save keeps those records exactly as they were on disk.
func TestLegacyRegistryListsAsPersistentAndSavesUnchanged(t *testing.T) {
	isolateConfig(t)
	path := useLegacyFixture(t)
	before := readRecords(t, path)

	var out bytes.Buffer
	if err := list(&out, ListOptions{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"borrowed-box", "do-box", "gcp-box"} {
		line := lineWith(out.String(), name)
		if !strings.Contains(line, " persistent ") {
			t.Errorf("%s does not list as persistent: %q", name, line)
		}
	}
	if strings.Contains(out.String(), "EXPIRED") {
		t.Errorf("a record with no expiry lists as expired:\n%s", out.String())
	}

	if err := SaveSandbox("new-box", &SandboxMeta{Provider: ProviderNone, IP: "203.0.113.20", Kind: KindOneOff}); err != nil {
		t.Fatal(err)
	}
	after := readRecords(t, path)
	for name, record := range before {
		if !reflect.DeepEqual(after[name], record) {
			t.Errorf("record %s changed on save:\nbefore %v\nafter  %v", name, record, after[name])
		}
	}
}

// lineWith returns the first line of s that starts with prefix.
func lineWith(s, prefix string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, prefix+" ") {
			return line
		}
	}
	return ""
}
