package sandbox

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if err := list(&out, io.Discard, ListOptions{}, time.Now()); err != nil {
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

// The create flags resolve to a kind and an expiry, and every bad
// combination stops before anything bills.
func TestResolveKind(t *testing.T) {
	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		oneOff   bool
		expires  string
		kind     string
		expected time.Time
		err      string
	}{
		{kind: KindPersistent},
		{expires: "3d", err: "needs --one-off"},
		{oneOff: true, kind: KindOneOff, expected: now.Add(48 * time.Hour)},
		{oneOff: true, expires: "36h", kind: KindOneOff, expected: now.Add(36 * time.Hour)},
		{oneOff: true, expires: "3d", kind: KindOneOff, expected: now.Add(72 * time.Hour)},
		{oneOff: true, expires: "2026-10-17", kind: KindOneOff, expected: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)},
		{oneOff: true, expires: "2026-10-17T09:30:00-04:00", kind: KindOneOff, expected: time.Date(2026, 10, 17, 13, 30, 0, 0, time.UTC)},
		{oneOff: true, expires: "soon", err: "invalid --expires"},
		{oneOff: true, expires: "2026-10-01", err: "not in the future"},
		{oneOff: true, expires: "-1h", err: "not in the future"},
		{oneOff: true, expires: "106752d", err: "too far away"},
	} {
		kind, expires, err := ResolveKind(tc.oneOff, tc.expires, now)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("ResolveKind(%v, %q) error = %v, want %q", tc.oneOff, tc.expires, err, tc.err)
			}
			continue
		}
		if err != nil || kind != tc.kind || !expires.Equal(tc.expected) || (!expires.IsZero() && expires.Location() != time.UTC) {
			t.Errorf("ResolveKind(%v, %q) = %q, %v, %v; want %q, %v", tc.oneOff, tc.expires, kind, expires, err, tc.kind, tc.expected)
		}
	}
}

// A purpose is one line of text: every control character is refused, and
// every other character, shell syntax included, is kept.
func TestValidatePurpose(t *testing.T) {
	for _, bad := range []string{"two\nlines", "a\ttab", "red \x1b[31m", "del \x7f", "next line \u0085"} {
		if err := ValidatePurpose(bad); err == nil || !strings.Contains(err.Error(), "control character") {
			t.Errorf("ValidatePurpose(%q) = %v, want a control character error", bad, err)
		}
	}
	if err := ValidatePurpose("fix \"login\" $(touch /tmp/x) `id`; rm -rf /tmp/y 'q' café"); err != nil {
		t.Errorf("ValidatePurpose refused a purpose without control characters: %v", err)
	}
}

// Label values obey GCP's rules on every provider, and the expiry label
// rounds up to the next day so a remote list never flags a sandbox early.
func TestSandboxLabels(t *testing.T) {
	got := sandboxLabels(&SandboxMeta{Kind: KindOneOff, CreatedBy: "Ops-Laptop.Example.LAN", ExpiresAt: "2026-10-17T15:00:00Z"})
	want := map[string]string{labelKind: "one-off", labelHost: "ops-laptop-example-lan", labelExpires: "2026-10-18"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("labels = %v, want %v", got, want)
	}
	if got := sandboxLabels(&SandboxMeta{ExpiresAt: "2026-10-17T00:00:00Z"})[labelExpires]; got != "2026-10-17" {
		t.Errorf("an expiry at midnight labels as %q, want 2026-10-17", got)
	}
	if got := sandboxLabels(&SandboxMeta{})[labelKind]; got != KindPersistent {
		t.Errorf("a record with no kind labels as %q, want persistent", got)
	}
	if got := labelValue(strings.Repeat("a", 70)); len(got) != 63 {
		t.Errorf("a long value is %d characters, want 63", len(got))
	}
}

// labelingProvider is a fake cloud that records the labels it gets and
// reports the machines it was given.
type labelingProvider struct {
	recordingProvider
	labels   map[string]string
	labelErr error
	machines []RemoteSandbox
	listErr  error
}

func (l *labelingProvider) Label(meta *SandboxMeta, labels map[string]string) error {
	l.labels = labels
	return l.labelErr
}

func (l *labelingProvider) ListLabeled() ([]RemoteSandbox, error) { return l.machines, l.listErr }

// stubHostname makes createdBy the same on every machine that runs a test.
func stubHostname(t *testing.T) {
	t.Helper()
	orig := hostname
	hostname = func() string { return "ops-laptop" }
	t.Cleanup(func() { hostname = orig })
}

// A one-off create writes the four fields to the record and labels the
// machine. A labeling failure is a warning and does not fail the create.
func TestCreateOneOffRecordsKindAndLabels(t *testing.T) {
	expires := time.Date(2026, 10, 17, 15, 0, 0, 0, time.UTC)
	for _, labelErr := range []error{nil, errors.New("tag quota reached")} {
		t.Run(fmt.Sprint(labelErr), func(t *testing.T) {
			isolateConfig(t)
			useTempStore(t)
			stubCreateEnv(t)
			stubHostname(t)

			p := &labelingProvider{recordingProvider: recordingProvider{ip: "9.9.9.9", dropletID: 7}, labelErr: labelErr}
			stderr := captureStderr(t)
			err := Create(p, CreateOptions{Name: "task", Kind: KindOneOff, ExpiresAt: expires, Purpose: "x"})
			out := stderr()
			// stubCreateEnv stops every create at the provisioning wait,
			// which runs after the labels.
			if err == nil || !strings.Contains(err.Error(), "waiting for provisioning") {
				t.Fatalf("create stopped at %v, want the provisioning wait", err)
			}

			got, _ := GetSandbox("task")
			if got == nil || got.Kind != KindOneOff || got.Purpose != "x" || got.ExpiresAt != "2026-10-17T15:00:00Z" || got.CreatedBy != "ops-laptop" {
				t.Fatalf("record = %+v, want the four kind fields", got)
			}
			want := map[string]string{labelKind: "one-off", labelHost: "ops-laptop", labelExpires: "2026-10-18"}
			if !reflect.DeepEqual(p.labels, want) {
				t.Errorf("labels = %v, want %v", p.labels, want)
			}
			if warned := strings.Contains(out, "Warning: could not label"); warned != (labelErr != nil) {
				t.Errorf("label warning printed = %v, want %v:\n%s", warned, labelErr != nil, out)
			}
		})
	}
}

// A machine you already have gets the four fields and the marker. It has no
// provider to label it, and that is not a warning or an error. The test runs
// the real equip step up to the credential command, so it fails if that
// command does not carry the marker of this record.
func TestCreateOneOffOnAnExistingMachine(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)
	stubHostname(t)

	key := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(key, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	origCheck := checkReady
	checkReady = func(meta *SandboxMeta, name string) error { return nil }
	t.Cleanup(func() { checkReady = origCheck })
	origFetch := fetchAgent
	fetchAgent = func(meta *SandboxMeta, name string, artifacts *releaseArtifacts) error { return nil }
	t.Cleanup(func() { fetchAgent = origFetch })
	var injected string
	origExec := sshExec
	sshExec = func(args []string, stdin string) ([]byte, error) {
		injected = sentScript(args, stdin)
		return nil, errors.New("stop here")
	}
	t.Cleanup(func() { sshExec = origExec })

	stderr := captureStderr(t)
	err := Create(nil, CreateOptions{Name: "borrowed", IP: "203.0.113.10", SSHKey: key, Kind: KindOneOff, ExpiresAt: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), Purpose: "x"})
	out := stderr()
	if err == nil || !strings.Contains(err.Error(), "stop here") {
		t.Fatalf("create stopped at %v, want the stubbed equip step", err)
	}

	got, _ := GetSandbox("borrowed")
	if got == nil || got.Kind != KindOneOff || got.Purpose != "x" || got.ExpiresAt != "2026-10-17T00:00:00Z" || got.CreatedBy != "ops-laptop" {
		t.Fatalf("record = %+v, want the four kind fields", got)
	}
	marker := markerContent(&SandboxMeta{Kind: KindOneOff, Purpose: "x", ExpiresAt: "2026-10-17T00:00:00Z", CreatedBy: "ops-laptop"})
	if !strings.Contains(injected, base64.StdEncoding.EncodeToString(marker)) {
		t.Errorf("the credential command does not write the marker %s:\n%s", marker, injected)
	}
	if strings.Contains(strings.ToLower(out), "label") {
		t.Errorf("an existing machine has no labels, but create said:\n%s", out)
	}
}

// list shows the kind and the expiry, --expired filters, and --all-hosts
// adds the labeled machines that no local record names. A provider that
// cannot be asked gets one line, and the list still prints.
func TestListExpiredAndAllHosts(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for name, meta := range map[string]*SandboxMeta{
		"agent-box":  {Provider: "digitalocean", DropletID: 1, Status: "active"},
		"old-task":   {Provider: "digitalocean", DropletID: 2, Status: "active", Kind: KindOneOff, ExpiresAt: "2026-10-19T00:00:00Z"},
		"fresh-task": {Provider: ProviderNone, IP: "203.0.113.10", Status: "active", Kind: KindOneOff, ExpiresAt: "2026-10-22T00:00:00Z"},
	} {
		if err := SaveSandbox(name, meta); err != nil {
			t.Fatal(err)
		}
	}

	do := &labelingProvider{machines: []RemoteSandbox{
		// The same droplet as the local old-task: not listed twice.
		{Name: "old-task", Meta: SandboxMeta{Provider: "digitalocean", DropletID: 2, Kind: KindOneOff}},
		{Name: "their-task", Meta: SandboxMeta{Provider: "digitalocean", DropletID: 3, Kind: KindOneOff, ExpiresAt: "2026-10-18T00:00:00Z", CreatedBy: "build-host"}},
		{Name: "their-agent", Meta: SandboxMeta{Provider: "digitalocean", DropletID: 4}},
	}}
	orig := providerByName
	providerByName = func(name string) (Provider, error) {
		switch name {
		case "digitalocean":
			return do, nil
		case ProviderGCP:
			return nil, errors.New("gcloud is not on PATH")
		}
		return nil, nil
	}
	t.Cleanup(func() { providerByName = orig })

	run := func(opts ListOptions) (string, string) {
		t.Helper()
		var out, errOut bytes.Buffer
		if err := list(&out, &errOut, opts, now); err != nil {
			t.Fatal(err)
		}
		return out.String(), errOut.String()
	}

	out, _ := run(ListOptions{})
	if !strings.Contains(lineWith(out, "old-task"), "EXPIRED 2026-10-19T00:00:00Z") {
		t.Errorf("old-task is not flagged expired:\n%s", out)
	}
	if line := lineWith(out, "fresh-task"); !strings.Contains(line, " one-off ") || strings.Contains(line, "EXPIRED") {
		t.Errorf("fresh-task row = %q", line)
	}
	if strings.Index(out, "agent-box") > strings.Index(out, "fresh-task") {
		t.Errorf("rows are not sorted by name:\n%s", out)
	}
	if strings.Contains(out, "their-") {
		t.Errorf("list without --all-hosts shows remote machines:\n%s", out)
	}

	out, _ = run(ListOptions{Expired: true})
	if lineWith(out, "old-task") == "" || lineWith(out, "fresh-task") != "" || lineWith(out, "agent-box") != "" {
		t.Errorf("--expired shows the wrong rows:\n%s", out)
	}

	out, errOut := run(ListOptions{AllHosts: true})
	if line := lineWith(out, "their-task"); !strings.Contains(line, " remote ") || !strings.Contains(line, "EXPIRED") {
		t.Errorf("their-task row = %q, want remote and expired", line)
	}
	if line := lineWith(out, "their-agent"); !strings.Contains(line, " persistent ") || !strings.Contains(line, " remote ") {
		t.Errorf("their-agent row = %q, want persistent and remote", line)
	}
	if strings.Count(out, "old-task") != 1 {
		t.Errorf("a remote machine that a local record names is listed twice:\n%s", out)
	}
	if errOut != "gcp: cannot query: gcloud is not on PATH\n" {
		t.Errorf("want one line for gcp, which cannot be asked, and none for none, which is not a cloud:\n%s", errOut)
	}

	out, _ = run(ListOptions{AllHosts: true, Expired: true})
	if lineWith(out, "their-task") == "" || lineWith(out, "their-agent") != "" || lineWith(out, "agent-box") != "" {
		t.Errorf("--all-hosts --expired shows the wrong rows:\n%s", out)
	}

	do.listErr = errors.New("401 Unauthorized")
	out, errOut = run(ListOptions{AllHosts: true})
	if !strings.Contains(errOut, "digitalocean: cannot query: 401 Unauthorized") || lineWith(out, "agent-box") == "" {
		t.Errorf("a failed provider must print one line and leave the local list:\nout:\n%s\nerr:\n%s", out, errOut)
	}
}

// DigitalOcean reports every droplet with the spekk-sandbox tag, across
// pages, and reads the kind fields from its tags.
func TestDOProviderListLabeled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/droplets" || r.URL.Query().Get("tag_name") != "spekk-sandbox" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		var droplets []map[string]any
		if r.URL.Query().Get("page") == "1" {
			for i := 0; i < 200; i++ {
				droplets = append(droplets, map[string]any{"id": 1000 + i, "name": fmt.Sprintf("spekk-filler-%d", i)})
			}
		} else {
			droplets = append(droplets, map[string]any{
				"id": 42, "name": "spekk-their-task", "created_at": "2026-10-15T08:00:00Z",
				"region":   map[string]any{"slug": "ams3"},
				"tags":     []string{"spekk-sandbox", "spekk-kind:one-off", "spekk-expires:2026-10-18", "spekk-host:build-host"},
				"networks": map[string]any{"v4": []map[string]any{{"ip_address": "198.51.100.9", "type": "public"}}},
			})
		}
		json.NewEncoder(w).Encode(map[string]any{"droplets": droplets})
	}))
	defer ts.Close()

	found, err := (&DOProvider{client: NewClientWithToken("tok", ts.URL)}).ListLabeled()
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 201 {
		t.Fatalf("found %d droplets, want 201 across two pages", len(found))
	}
	got := found[200]
	want := RemoteSandbox{Name: "their-task", Meta: SandboxMeta{Provider: "digitalocean", DropletID: 42, IP: "198.51.100.9", Region: "ams3", CreatedAt: "2026-10-15T08:00:00Z", Kind: KindOneOff, ExpiresAt: "2026-10-18T00:00:00Z", CreatedBy: "build-host"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("droplet = %+v, want %+v", got, want)
	}
}

// GCP reports the labeled VMs in the default project.
func TestGCPProviderListLabeled(t *testing.T) {
	fakeGcloud(t, func(cmd string) (string, error) {
		switch cmd {
		case "config get-value project":
			return "proj-a", nil
		case "compute instances list --project proj-a --filter labels.spekk-kind:* --format json":
			return `[{"name": "spekk-their-task", "zone": "https://www.googleapis.com/compute/v1/projects/proj-a/zones/us-east1-b",
				"creationTimestamp": "2026-10-15T01:00:00.000-07:00",
				"labels": {"spekk-kind": "one-off", "spekk-expires": "2026-10-18", "spekk-host": "build-host"},
				"networkInterfaces": [{"accessConfigs": [{"natIP": "192.0.2.50"}]}]}]`, nil
		}
		return "", fmt.Errorf("unexpected gcloud %s", cmd)
	})
	found, err := (&GCPProvider{}).ListLabeled()
	if err != nil {
		t.Fatal(err)
	}
	want := []RemoteSandbox{{Name: "their-task", Meta: SandboxMeta{Provider: ProviderGCP, GCPInstance: "spekk-their-task", Project: "proj-a", Region: "us-east1-b", IP: "192.0.2.50", CreatedAt: "2026-10-15T08:00:00Z", Kind: KindOneOff, ExpiresAt: "2026-10-18T00:00:00Z", CreatedBy: "build-host"}}}
	if !reflect.DeepEqual(found, want) {
		t.Errorf("found %+v, want %+v", found, want)
	}
}
