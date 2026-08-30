package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

// isolateConfig points the config dir at a temp dir for one test.
//
// HOME has to move with XDG_CONFIG_HOME. GlobalConfigDir migrates an old
// $HOME/.spekk into the new location by renaming it, so a test that moves
// only XDG_CONFIG_HOME renames the developer's real config directory into
// a t.TempDir() that is deleted when the test ends.
func isolateConfig(t *testing.T) {
	t.Helper()
	config.ResetCacheForTest(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// stubCreateEnv supplies the environment and release artifacts Create needs,
// and short-circuits the ten-minute provisioning wait.
func stubCreateEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"} {
		t.Setenv(v, "x")
	}
	origArtifacts := fetchArtifacts
	fetchArtifacts = func(tag string) (*releaseArtifacts, error) {
		return &releaseArtifacts{Version: "test", CloudInit: []byte("#cloud-config")}, nil
	}
	t.Cleanup(func() { fetchArtifacts = origArtifacts })

	origWait := waitReady
	waitReady = func(ip, keyPath, name string) error { return fmt.Errorf("boom") }
	t.Cleanup(func() { waitReady = origWait })
}

// useTempStore points the metadata file at a temp dir and returns its path.
func useTempStore(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sandboxes.json")
	orig := sandboxesFile
	sandboxesFile = func() string { return path }
	t.Cleanup(func() { sandboxesFile = orig })
	return path
}

// legacyStore writes a sandboxes.json in the shape an older binary wrote:
// a droplet id and an ssh key id, and no provider field at all.
const legacyStore = `{
  "old-box": {
    "dropletId": 558093268,
    "sshKeyId": 4433221,
    "ip": "1.2.3.4",
    "region": "nyc1",
    "size": "s-2vcpu-4gb",
    "createdAt": "2026-01-01T00:00:00Z",
    "status": "active"
  }
}`

// A sandbox created before the provider field existed must still be fully
// destroyable. The failure this pins is silent: teardown is skipped, the
// metadata is removed anyway, and the droplet bills forever with no local
// record of its id.
func TestDestroyLegacyMetadataDeletesDroplet(t *testing.T) {
	isolateConfig(t)
	path := useTempStore(t)
	if err := os.WriteFile(path, []byte(legacyStore), 0o600); err != nil {
		t.Fatal(err)
	}

	var dropletDeleted, keyDeleted bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/droplets/558093268":
			dropletDeleted = true
		case "/v2/account/keys/4433221":
			keyDeleted = true
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer ts.Close()

	meta, err := GetSandbox("old-box")
	if err != nil || meta == nil {
		t.Fatalf("legacy entry must load: %v", err)
	}
	if meta.DropletID != 558093268 {
		t.Fatalf("legacy dropletId lost on load: %+v", meta)
	}

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	if err := Destroy(p, "old-box", true); err != nil {
		t.Fatal(err)
	}
	if !dropletDeleted {
		t.Error("droplet was never deleted; it is orphaned and still billing")
	}
	if !keyDeleted {
		t.Error("DO ssh key was never deleted")
	}
	if got, _ := GetSandbox("old-box"); got != nil {
		t.Error("metadata should be gone after a successful destroy")
	}
}

// An entry with no provider field is DigitalOcean's, because that was the
// only provider when such entries were written.
func TestProviderFromMetaReadsEmptyAsDigitalOcean(t *testing.T) {
	isolateConfig(t)
	t.Setenv("DO_API_TOKEN", "tok")
	p, err := ProviderFromMeta(&SandboxMeta{DropletID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "digitalocean" {
		t.Errorf("Name = %q, want digitalocean", p.Name())
	}
	if _, err := ProviderFromMeta(&SandboxMeta{Provider: "nowhere"}); err == nil {
		t.Error("an unknown provider must be an error, not a default")
	}
}

// Writing any entry must not rewrite the others into a shape that has lost
// their droplet ids. That loss is unrecoverable: it is the identifier a
// later destroy needs.
func TestSaveKeepsLegacyFieldsOfOtherEntries(t *testing.T) {
	isolateConfig(t)
	path := useTempStore(t)
	if err := os.WriteFile(path, []byte(legacyStore), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SaveSandbox("new-box", &SandboxMeta{Provider: "digitalocean", DropletID: 7, IP: "5.6.7.8"}); err != nil {
		t.Fatal(err)
	}

	var raw map[string]map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["old-box"]["dropletId"] != float64(558093268) {
		t.Errorf("dropletId of the untouched entry was lost: %v", raw["old-box"])
	}
	if raw["old-box"]["sshKeyId"] != float64(4433221) {
		t.Errorf("sshKeyId of the untouched entry was lost: %v", raw["old-box"])
	}
}

// Destroy must not delete a key it did not generate, and must stop rather
// than remove the metadata that makes an orphan findable.
func TestDestroyGuards(t *testing.T) {
	t.Run("provider refuses when no machine is recorded", func(t *testing.T) {
		p := &DOProvider{client: NewClientWithToken("tok", "http://127.0.0.1:1")}
		err := p.Destroy(&SandboxMeta{IP: "1.2.3.4"})
		if !errors.Is(err, ErrNoMachineRecorded) {
			t.Fatalf("want ErrNoMachineRecorded, got %v", err)
		}
	})

	t.Run("a failed teardown keeps the metadata", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		if err := SaveSandbox("stuck", &SandboxMeta{Provider: "digitalocean", DropletID: 5}); err != nil {
			t.Fatal(err)
		}
		p := &recordingProvider{destroyErr: fmt.Errorf("DO API is down")}
		// Even --force must not clear a record whose machine may live.
		if err := Destroy(p, "stuck", true); err == nil {
			t.Fatal("a failed teardown must fail the command")
		}
		if got, _ := GetSandbox("stuck"); got == nil {
			t.Error("metadata must survive, or the machine becomes unfindable")
		}
	})

	t.Run("force clears a record that names no machine", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		if err := SaveSandbox("ghost", &SandboxMeta{Provider: "digitalocean", IP: "1.2.3.4"}); err != nil {
			t.Fatal(err)
		}
		p := &recordingProvider{destroyErr: fmt.Errorf("%w: gone", ErrNoMachineRecorded)}
		if err := Destroy(p, "ghost", true); err != nil {
			t.Fatalf("--force must clear a record naming no machine: %v", err)
		}
		if got, _ := GetSandbox("ghost"); got != nil {
			t.Error("the record should be gone")
		}
	})

	t.Run("keeps a key it did not generate", func(t *testing.T) {
		isolateConfig(t)
		home := os.Getenv("XDG_CONFIG_HOME")

		operatorKey := filepath.Join(t.TempDir(), "id_ed25519")
		for _, p := range []string{operatorKey, operatorKey + ".pub"} {
			if err := os.WriteFile(p, []byte("k"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if ownsKeyPair(operatorKey) {
			t.Fatal("a key outside the generated keys dir is not ours to delete")
		}
		generated := filepath.Join(home, "spekk", "keys", "box")
		if !ownsKeyPair(generated) {
			t.Fatalf("a key under %s is ours", generatedKeysDir())
		}
	})
}

func TestDOProviderDestroy(t *testing.T) {
	var dropletDeleted, sshKeyDeleted bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "DELETE" && r.URL.Path == "/v2/droplets/100":
			dropletDeleted = true
			w.WriteHeader(204)
		case r.Method == "DELETE" && r.URL.Path == "/v2/account/keys/200":
			sshKeyDeleted = true
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	if err := p.Destroy(&SandboxMeta{DropletID: 100, SSHKeyID: 200}); err != nil {
		t.Fatal(err)
	}
	if !dropletDeleted || !sshKeyDeleted {
		t.Errorf("droplet deleted=%v, key deleted=%v", dropletDeleted, sshKeyDeleted)
	}
}

// A droplet the console already removed is not an error worth failing on;
// the local cleanup still has to run.
func TestDOProviderDestroyHandles404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{"message": "not found"})
	}))
	defer ts.Close()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	if err := p.Destroy(&SandboxMeta{DropletID: 1, SSHKeyID: 2}); err != nil {
		t.Fatalf("Destroy should handle 404 gracefully, got: %s", err)
	}
}

func TestDOProviderStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/droplets/555" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"droplet": map[string]any{"id": 555, "status": "active"},
		})
	}))
	defer ts.Close()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	status, err := p.Status(&SandboxMeta{DropletID: 555})
	if err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Errorf("Status = %q, want %q", status, "active")
	}
}

// Create records what it resolved, not what the flags said. Omitting
// --region and --size must still leave a region and a size in metadata.
func TestDOProviderCreateFillsMetaWithResolvedValues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/v2/account/keys":
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{
				"ssh_key": map[string]any{"id": 10, "name": "test", "fingerprint": "aa:bb"},
			})
		case r.Method == "GET" && r.URL.Path == "/v2/account/keys":
			json.NewEncoder(w).Encode(map[string]any{
				"ssh_keys": []map[string]any{{"id": 10, "name": "test"}},
			})
		case r.Method == "POST" && r.URL.Path == "/v2/droplets":
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)
			if req["region"] != "nyc1" || req["size"] != "s-2vcpu-4gb" {
				t.Errorf("defaults not sent to the API: %v", req)
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{
				"droplet": map[string]any{"id": 42, "name": "spekk-test", "status": "active"},
			})
		case r.Method == "GET" && r.URL.Path == "/v2/droplets/42":
			json.NewEncoder(w).Encode(map[string]any{
				"droplet": map[string]any{
					"id": 42, "status": "active",
					"networks": map[string]any{
						"v4": []map[string]any{{"ip_address": "5.6.7.8", "type": "public"}},
					},
				},
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	isolateConfig(t)

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	meta := &SandboxMeta{}
	if err := p.Create("testbox", CreateOptions{Name: "testbox"}, meta); err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Remove(meta.SSHKeyPath)
		os.Remove(meta.SSHKeyPath + ".pub")
	}()

	if meta.Region != "nyc1" || meta.Size != "s-2vcpu-4gb" {
		t.Errorf("resolved defaults not recorded: region=%q size=%q", meta.Region, meta.Size)
	}
	if meta.DropletID != 42 || meta.SSHKeyID != 10 {
		t.Errorf("identifiers not recorded: %+v", meta)
	}
	if meta.IP != "5.6.7.8" || meta.SSHKeyPath == "" {
		t.Errorf("connection details not recorded: %+v", meta)
	}
	if !ownsKeyPair(meta.SSHKeyPath) {
		t.Errorf("a generated key must be recognized as ours: %s", meta.SSHKeyPath)
	}
}

// A missing API token must yield a nil Provider, not a nil *DOProvider
// wrapped in a non-nil interface. Callers that degrade to the stored data
// check for nil, and a typed nil walks straight past that check.
func TestProviderFromMetaReturnsUntypedNilOnError(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	t.Setenv("DO_API_TOKEN", "")
	t.Setenv("DIGITALOCEAN_TOKEN", "")

	p, err := ProviderFromMeta(&SandboxMeta{DropletID: 1})
	if err == nil {
		t.Fatal("a missing token must be an error")
	}
	if p != nil {
		t.Fatalf("provider must be nil, got a non-nil interface holding %T", p)
	}
	// The nil must survive the fallback path Status takes.
	if err := Status(p, "anything"); err == nil {
		t.Log("Status returned no error, which is fine; the point is that it did not panic")
	}
}

// Create records the machine as soon as it exists. Without this, a failure
// during provisioning leaves a running droplet that `destroy` cannot find.
func TestCreateRecordsMachineBeforeProvisioning(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)

	// Fail at the first step after the machine exists — the case the
	// metadata entry is there for.
	p := &recordingProvider{ip: "9.9.9.9", dropletID: 4242}
	if err := Create(p, CreateOptions{Name: "halfway"}); err == nil {
		t.Fatal("expected create to fail after the machine was made")
	}
	got, err := GetSandbox("halfway")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a machine was created but nothing was recorded; destroy cannot reach it")
	}
	if got.DropletID != 4242 || got.IP != "9.9.9.9" {
		t.Errorf("recorded the wrong machine: %+v", got)
	}
	if got.Status != "provisioning" {
		t.Errorf("status = %q, want provisioning", got.Status)
	}
}

type recordingProvider struct {
	ip         string
	dropletID  int
	createErr  error
	destroyErr error
}

func (r *recordingProvider) Name() string { return "digitalocean" }

func (r *recordingProvider) Create(name string, opts CreateOptions, meta *SandboxMeta) error {
	meta.IP = r.ip
	meta.DropletID = r.dropletID
	meta.Region = "nyc1"
	meta.Size = "s-2vcpu-4gb"
	meta.SSHKeyPath = filepath.Join(generatedKeysDir(), name)
	return r.createErr
}

func (r *recordingProvider) Destroy(meta *SandboxMeta) error          { return r.destroyErr }
func (r *recordingProvider) Status(meta *SandboxMeta) (string, error) { return "active", nil }

// A provider that makes a machine and then fails must still leave the
// machine named on disk. Otherwise only stderr holds its identifier.
func TestCreateRecordsMachineWhenTheProviderFails(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)

	p := &recordingProvider{ip: "9.9.9.9", dropletID: 777, createErr: fmt.Errorf("timed out waiting for an address")}
	if err := Create(p, CreateOptions{Name: "halfmade"}); err == nil {
		t.Fatal("expected the provider error to surface")
	}
	got, _ := GetSandbox("halfmade")
	if got == nil {
		t.Fatal("the machine the provider made is unrecorded, so destroy cannot reach it")
	}
	if got.DropletID != 777 {
		t.Errorf("recorded %+v, want droplet 777", got)
	}
}

// Re-running create after a failure must not overwrite the record of the
// machine that failure left running.
func TestCreateRefusesAnExistingName(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)

	if err := SaveSandbox("taken", &SandboxMeta{Provider: "digitalocean", DropletID: 100, IP: "1.2.3.4"}); err != nil {
		t.Fatal(err)
	}
	err := Create(&recordingProvider{ip: "2.2.2.2", dropletID: 200}, CreateOptions{Name: "taken"})
	if err == nil {
		t.Fatal("create must refuse a name that is already recorded")
	}
	if !strings.Contains(err.Error(), "destroy") {
		t.Errorf("the error should say how to clear it, got %q", err)
	}
	got, _ := GetSandbox("taken")
	if got == nil || got.DropletID != 100 {
		t.Errorf("the original record must survive, got %+v", got)
	}
}
