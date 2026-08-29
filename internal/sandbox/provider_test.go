package sandbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// --- Provider interface conformance ---

// Verify DOProvider satisfies the Provider interface at compile time.
var _ Provider = (*DOProvider)(nil)

// --- SandboxMeta field tests ---

func TestSandboxMetaHasProviderAndInstanceID(t *testing.T) {
	tmpDir := t.TempDir()
	origFile := sandboxesFile
	sandboxesFile = func() string {
		return filepath.Join(tmpDir, "sandboxes.json")
	}
	defer func() { sandboxesFile = origFile }()

	meta := &SandboxMeta{
		Provider:   "digitalocean",
		InstanceID: `{"droplet_id":42,"ssh_key_id":7}`,
		IP:         "10.0.0.1",
		SSHKeyPath: "/tmp/key",
	}
	if err := SaveSandbox("test", meta); err != nil {
		t.Fatal(err)
	}

	got, err := GetSandbox("test")
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "digitalocean" {
		t.Errorf("Provider = %q, want %q", got.Provider, "digitalocean")
	}
	if got.InstanceID != `{"droplet_id":42,"ssh_key_id":7}` {
		t.Errorf("InstanceID = %q, want JSON with droplet_id/ssh_key_id", got.InstanceID)
	}
}

func TestSandboxMetaJSON(t *testing.T) {
	meta := &SandboxMeta{
		Provider:   "manual",
		InstanceID: "my-box",
		IP:         "192.168.1.1",
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	if decoded["provider"] != "manual" {
		t.Errorf("JSON provider = %v", decoded["provider"])
	}
	if decoded["instanceId"] != "my-box" {
		t.Errorf("JSON instanceId = %v", decoded["instanceId"])
	}
	// Verify old fields are absent.
	if _, ok := decoded["dropletId"]; ok {
		t.Error("JSON should not contain dropletId")
	}
	if _, ok := decoded["sshKeyId"]; ok {
		t.Error("JSON should not contain sshKeyId")
	}
}

// --- DOProvider tests ---

func TestDOProviderDestroy(t *testing.T) {
	dropletDeleted := false
	sshKeyDeleted := false

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
	instanceID := encodeInstanceState(doInstanceState{DropletID: 100, SSHKeyID: 200})

	if err := p.Destroy(instanceID); err != nil {
		t.Fatal(err)
	}
	if !dropletDeleted {
		t.Error("expected droplet DELETE")
	}
	if !sshKeyDeleted {
		t.Error("expected SSH key DELETE")
	}
}

func TestDOProviderStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/droplets/555" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"droplet": map[string]interface{}{"id": 555, "status": "active"},
		})
	}))
	defer ts.Close()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	instanceID := encodeInstanceState(doInstanceState{DropletID: 555})

	status, err := p.Status(instanceID)
	if err != nil {
		t.Fatal(err)
	}
	if status != "active" {
		t.Errorf("Status = %q, want %q", status, "active")
	}
}

func TestDOProviderDestroyHandles404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "not found",
		})
	}))
	defer ts.Close()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	instanceID := encodeInstanceState(doInstanceState{DropletID: 1, SSHKeyID: 2})

	// 404s are handled gracefully, not returned as errors.
	if err := p.Destroy(instanceID); err != nil {
		t.Fatalf("Destroy should handle 404 gracefully, got: %s", err)
	}
}

func TestInstanceIDIsOpaque(t *testing.T) {
	// The generic layer stores InstanceID as a string without interpreting it.
	tmpDir := t.TempDir()
	origFile := sandboxesFile
	sandboxesFile = func() string {
		return filepath.Join(tmpDir, "sandboxes.json")
	}
	defer func() { sandboxesFile = origFile }()

	opaqueID := `{"droplet_id":999,"ssh_key_id":888}`
	meta := &SandboxMeta{
		Provider:   "digitalocean",
		InstanceID: opaqueID,
		IP:         "1.2.3.4",
	}
	SaveSandbox("opaque-test", meta)

	got, _ := GetSandbox("opaque-test")
	if got.InstanceID != opaqueID {
		t.Errorf("InstanceID should be preserved verbatim, got %q", got.InstanceID)
	}

	// Only the provider can decode it.
	state, err := decodeInstanceState(got.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if state.DropletID != 999 || state.SSHKeyID != 888 {
		t.Errorf("decoded state = %+v, want {999 888}", state)
	}
}

func TestCreateAcceptsProvider(t *testing.T) {
	// Verify that Create, Destroy, and Status accept a Provider parameter.
	// This is a compile-time check — the functions exist with the right signatures.
	var p Provider = &stubProvider{}
	_ = p

	// These type assertions verify the function signatures at compile time.
	var _ func(Provider, CreateOptions) error = Create
	var _ func(Provider, string, bool) error = Destroy
	var _ func(Provider, string) error = Status
}

// stubProvider is a no-op Provider for compile-time checks.
type stubProvider struct{}

func (s *stubProvider) Create(name string, config map[string]string) (*CreateResult, error) {
	return &CreateResult{IP: "127.0.0.1", InstanceID: "stub", Provider: "stub"}, nil
}
func (s *stubProvider) Destroy(instanceID string) error     { return nil }
func (s *stubProvider) Status(instanceID string) (string, error) { return "active", nil }

func TestEncodeDecodeInstanceState(t *testing.T) {
	original := doInstanceState{DropletID: 12345, SSHKeyID: 67890}
	encoded := encodeInstanceState(original)
	decoded, err := decodeInstanceState(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.DropletID != 12345 || decoded.SSHKeyID != 67890 {
		t.Errorf("round-trip failed: got %+v", decoded)
	}
}

func TestDecodeInstanceStateInvalid(t *testing.T) {
	_, err := decodeInstanceState("not-json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCreateResultProvider(t *testing.T) {
	// Verify CreateResult carries provider name.
	r := &CreateResult{
		IP:         "10.0.0.1",
		InstanceID: "abc",
		SSHKeyPath: "/tmp/key",
		Provider:   "digitalocean",
	}
	if r.Provider != "digitalocean" {
		t.Errorf("Provider = %q", r.Provider)
	}
}

func TestSandboxMetaNoDropletIDField(t *testing.T) {
	// Serialize and verify DropletID / SSHKeyID fields don't exist.
	meta := &SandboxMeta{Provider: "digitalocean", InstanceID: "x"}
	data, _ := json.Marshal(meta)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	for _, field := range []string{"dropletId", "sshKeyId"} {
		if _, exists := raw[field]; exists {
			t.Errorf("SandboxMeta JSON should not contain %q", field)
		}
	}
	// Verify new fields are present.
	if raw["provider"] != "digitalocean" {
		t.Error("missing provider field")
	}
	if raw["instanceId"] != "x" {
		t.Error("missing instanceId field")
	}
}

func TestDOProviderCreateResult(t *testing.T) {
	// Test that DOProvider.Create returns a CreateResult with correct fields.
	// Uses a mock server that handles SSH key upload + droplet creation.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/v2/account/keys":
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ssh_key": map[string]interface{}{"id": 10, "name": "test", "fingerprint": "aa:bb"},
			})
		case r.Method == "GET" && r.URL.Path == "/v2/account/keys":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ssh_keys": []map[string]interface{}{
					{"id": 10, "name": "test"},
				},
			})
		case r.Method == "POST" && r.URL.Path == "/v2/droplets":
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"droplet": map[string]interface{}{
					"id": 42, "name": "spekk-test", "status": "active",
					"networks": map[string]interface{}{
						"v4": []map[string]interface{}{
							{"ip_address": "5.6.7.8", "type": "public"},
						},
					},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v2/droplets/42":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"droplet": map[string]interface{}{
					"id": 42, "status": "active",
					"networks": map[string]interface{}{
						"v4": []map[string]interface{}{
							{"ip_address": "5.6.7.8", "type": "public"},
						},
					},
				},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	// Set up temp key directory so generateSSHKeyPair succeeds.
	tmpDir := t.TempDir()
	origKeysDir := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if origKeysDir != "" {
			os.Setenv("XDG_CONFIG_HOME", origKeysDir)
		}
	}()

	p := &DOProvider{client: NewClientWithToken("tok", ts.URL)}
	result, err := p.Create("testbox", map[string]string{
		"region": "nyc1",
		"size":   "s-1vcpu-1gb",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "digitalocean" {
		t.Errorf("Provider = %q, want digitalocean", result.Provider)
	}
	if result.IP != "5.6.7.8" {
		t.Errorf("IP = %q, want 5.6.7.8", result.IP)
	}
	if result.InstanceID == "" {
		t.Error("InstanceID should not be empty")
	}
	if result.SSHKeyPath == "" {
		t.Error("SSHKeyPath should not be empty")
	}

	// Verify InstanceID encodes DO-specific state.
	var state doInstanceState
	if err := json.Unmarshal([]byte(result.InstanceID), &state); err != nil {
		t.Fatalf("InstanceID should be valid JSON: %s", err)
	}
	if state.DropletID != 42 {
		t.Errorf("DropletID = %d, want 42", state.DropletID)
	}
	if state.SSHKeyID != 10 {
		t.Errorf("SSHKeyID = %d, want 10", state.SSHKeyID)
	}

	// Cleanup generated key files.
	os.Remove(result.SSHKeyPath)
	os.Remove(result.SSHKeyPath + ".pub")
}
