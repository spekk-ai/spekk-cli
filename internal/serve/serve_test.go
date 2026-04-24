package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListenOnPort(t *testing.T) {
	ln1, err := listenOnPort("127.0.0.1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()

	if ln1 == nil {
		t.Fatal("expected non-nil listener")
	}
}

func TestListenOnPortRetry(t *testing.T) {
	// Bind a port
	ln1, err := listenOnPort("127.0.0.1", 14118, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln1.Close()

	// Try same port with retry
	ln2, err := listenOnPort("127.0.0.1", 14118, 5)
	if err != nil {
		t.Fatal(err)
	}
	defer ln2.Close()

	if ln1.Addr().String() == ln2.Addr().String() {
		t.Error("should bind to different port")
	}
}

func TestHealthEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestOutgoingMessageJSON(t *testing.T) {
	msg := outgoingMessage{
		Event: "coach:status",
		Data:  statusData{State: "thinking"},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["event"] != "coach:status" {
		t.Errorf("expected coach:status event, got %v", parsed["event"])
	}

	dataMap, ok := parsed["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be object")
	}
	if dataMap["state"] != "thinking" {
		t.Errorf("expected thinking state, got %v", dataMap["state"])
	}
}

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{"empty origin", "", true},
		{"localhost", "http://localhost:3000", true},
		{"localhost no port", "http://localhost", true},
		{"ipv4 loopback", "http://127.0.0.1:8080", true},
		{"ipv6 loopback", "http://[::1]:8080", true},
		{"chrome extension", "chrome-extension://abcdef123456", true},
		{"external origin", "http://evil.com", false},
		{"https external", "https://attacker.example.com", false},
		{"similar name", "http://localhost.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			got := checkOrigin(r)
			if got != tt.want {
				t.Errorf("checkOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestAssistantDataJSON(t *testing.T) {
	msg := outgoingMessage{
		Event: "coach:assistant",
		Data:  assistantData{Content: "Hello!", SessionID: "s1"},
	}
	data, _ := json.Marshal(msg)

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	dataMap := parsed["data"].(map[string]interface{})
	if dataMap["content"] != "Hello!" {
		t.Errorf("expected content Hello!, got %v", dataMap["content"])
	}
	if dataMap["session_id"] != "s1" {
		t.Errorf("expected session_id s1, got %v", dataMap["session_id"])
	}
}
