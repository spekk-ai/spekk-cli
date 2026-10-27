package appclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientRefusesCredentialForwarding(t *testing.T) {
	for _, host := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com/path", "https://example.com?", "https://example.com/#fragment", "https://:443"} {
		if _, err := ValidateHost(host); err == nil {
			t.Fatalf("accepted unsafe host %q", host)
		}
	}
	var targetAuth string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, err := New(Config{Host: source.URL, Key: "private-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(context.Background(), "GET", "/api/sandbox-launches/", nil); err == nil {
		t.Fatal("redirect was accepted")
	}
	if targetAuth != "" {
		t.Fatalf("credential reached redirect target: %q", targetAuth)
	}
}

func TestClientReportsStatusWithoutKey(t *testing.T) {
	for _, status := range []int{401, 403, 404, 409, 429, 503} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"detail":"private-key denied"}`))
		}))
		client, err := New(Config{Host: server.URL, Key: "private-key"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Do(context.Background(), "POST", "/api/sandbox-launches/", map[string]any{})
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.Status != status || strings.Contains(err.Error(), "private-key") {
			t.Fatalf("status %d: unsafe or wrong error: %v", status, err)
		}
		if status == 401 && !strings.Contains(err.Error(), "new personal API key") {
			t.Fatalf("401 lacks replacement instruction: %v", err)
		}
		server.Close()
	}
}

func TestCredentialFileIsPrivateInASharedConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	// Other spekk commands make this directory with mode 0755.
	if err := os.Mkdir(filepath.Join(home, "spekk"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfig("https://app.example.test", "private-key\n"); err != nil {
		t.Fatal(err)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig()
	if err != nil || config.Key != "private-key" {
		t.Fatalf("could not read saved key: %v", err)
	}
	if err := SaveOperation(Operation{Kind: "send", RequestID: "request-1", Host: "https://app.example.test"}); err != nil {
		t.Fatalf("could not save an operation record: %v", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("accepted a group-readable key")
	}
}
