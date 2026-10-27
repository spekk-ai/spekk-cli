package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// testAPIToken is the API token every fake control host accepts.
const testAPIToken = "api-token-for-tests"

// recordedRequest is one request the fake control host received, whole, so a
// test can search every part of it for a value that must not be there.
type recordedRequest struct {
	Method, Path, RawQuery string
	Header                 http.Header
	Body                   string
}

// fakeControlHost follows the control host contract in
// specs/sandbox-self-register/: register, revoke, and the schedule routes.
type fakeControlHost struct {
	t   *testing.T
	srv *httptest.Server

	mu        sync.Mutex
	requests  []recordedRequest
	nextID    int
	agents    map[string]*fakeAgent
	schedules map[string][]map[string]any // by agent id, in slug order

	// pageSize splits a schedule list into pages, as the real paginator does.
	pageSize int
	// revokeStatus, when set, is the answer to every revoke.
	revokeStatus int
	// triggerStatus, when set, is the answer to every trigger.
	triggerStatus int
}

type fakeAgent struct {
	Name, SandboxID, Organization, Token string
	Revoked                              bool
}

// useControlHost starts a fake control host and points the environment at it.
func useControlHost(t *testing.T) *fakeControlHost {
	t.Helper()
	f := &fakeControlHost{t: t, agents: map[string]*fakeAgent{}, schedules: map[string][]map[string]any{}, pageSize: 25}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	t.Setenv("SPEKK_HOST", f.srv.URL)
	t.Setenv(apiTokenVar, testAPIToken)
	return f
}

// addAgent registers an agent as if another command had, and returns its id.
func (f *fakeControlHost) addAgent(name string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.nextID)
	f.agents[id] = &fakeAgent{Name: name, SandboxID: "spekk-" + name, Token: "agent-token-" + id}
	return id
}

// calls lists "METHOD path" for each request, in order.
func (f *fakeControlHost) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.requests {
		out = append(out, r.Method+" "+r.Path)
	}
	return out
}

func (f *fakeControlHost) agent(id string) *fakeAgent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.agents[id]
}

func (f *fakeControlHost) reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		json.NewEncoder(w).Encode(body)
	}
}

func (f *fakeControlHost) serve(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, recordedRequest{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Header: r.Header.Clone(), Body: string(data)})

	if r.Header.Get("Authorization") != "Token "+testAPIToken {
		f.reply(w, 401, map[string]string{"detail": "Authentication credentials were not provided."})
		return
	}
	var body map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &body); err != nil {
			f.reply(w, 400, map[string]string{"detail": "JSON parse error"})
			return
		}
	}

	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/"), "/")
	if parts[0] == "" {
		parts = nil
	}
	switch {
	case len(parts) == 0 && r.Method == http.MethodPost:
		f.register(w, body)
	case len(parts) == 2 && parts[1] == "revoke" && r.Method == http.MethodPost:
		a := f.agents[parts[0]]
		if a == nil {
			f.reply(w, 404, map[string]string{"detail": "No Agent matches the given query."})
			return
		}
		if f.revokeStatus != 0 {
			f.reply(w, f.revokeStatus, map[string]string{"detail": "revoke refused by the test"})
			return
		}
		a.Revoked = true
		f.reply(w, 200, map[string]any{"id": parts[0], "revoked": true})
	case len(parts) >= 2 && parts[1] == "schedules":
		if f.agents[parts[0]] == nil {
			f.reply(w, 404, map[string]string{"detail": "Not found."})
			return
		}
		f.serveSchedules(w, r, parts[0], parts[2:], body)
	default:
		f.reply(w, 404, map[string]string{"detail": "Not found."})
	}
}

func (f *fakeControlHost) register(w http.ResponseWriter, body map[string]any) {
	name, _ := body["name"].(string)
	sandboxID, _ := body["sandbox_id"].(string)
	if name == "" || sandboxID == "" {
		f.reply(w, 400, map[string][]string{"name": {"This field is required."}})
		return
	}
	for _, a := range f.agents {
		if a.SandboxID == sandboxID {
			f.reply(w, 400, map[string][]string{"sandbox_id": {"agent with this sandbox id already exists."}})
			return
		}
	}
	f.nextID++
	id := fmt.Sprintf("00000000-0000-4000-8000-%012d", f.nextID)
	org, _ := body["organization"].(string)
	a := &fakeAgent{Name: name, SandboxID: sandboxID, Organization: org, Token: "agent-token-" + id}
	f.agents[id] = a
	f.reply(w, 201, map[string]any{"id": id, "name": name, "sandbox_id": sandboxID, "status": "offline", "auth_token": a.Token})
}

func (f *fakeControlHost) serveSchedules(w http.ResponseWriter, r *http.Request, agentID string, rest []string, body map[string]any) {
	list := f.schedules[agentID]
	find := func(id string) int {
		for i, s := range list {
			if s["id"] == id {
				return i
			}
		}
		return -1
	}
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		start, end := (page-1)*f.pageSize, page*f.pageSize
		if start > len(list) {
			f.reply(w, 404, map[string]string{"detail": "Invalid page."})
			return
		}
		end = min(end, len(list))
		var next any
		if end < len(list) {
			next = fmt.Sprintf("%s/api/agents/%s/schedules/?page=%d", f.srv.URL, agentID, page+1)
		}
		f.reply(w, 200, map[string]any{"count": len(list), "next": next, "previous": nil, "results": list[start:end]})
	case len(rest) == 0 && r.Method == http.MethodPost:
		f.nextID++
		s := map[string]any{"id": fmt.Sprintf("10000000-0000-4000-8000-%012d", f.nextID), "agent": agentID, "timezone": "UTC", "role": "builder", "enabled": true, "custom_context": "", "next_fire_at": "2026-10-16T09:00:00Z"}
		for k, v := range body {
			s[k] = v
		}
		f.schedules[agentID] = append(list, s)
		f.reply(w, 201, s)
	case len(rest) == 1 && r.Method == http.MethodPatch:
		i := find(rest[0])
		if i < 0 {
			f.reply(w, 404, map[string]string{"detail": "Not found."})
			return
		}
		for k, v := range body {
			list[i][k] = v
		}
		f.reply(w, 200, list[i])
	case len(rest) == 1 && r.Method == http.MethodDelete:
		i := find(rest[0])
		if i < 0 {
			f.reply(w, 404, map[string]string{"detail": "Not found."})
			return
		}
		f.schedules[agentID] = append(list[:i], list[i+1:]...)
		w.WriteHeader(204)
	case len(rest) == 2 && rest[1] == "trigger" && r.Method == http.MethodPost:
		if find(rest[0]) < 0 {
			f.reply(w, 404, map[string]string{"detail": "Not found."})
			return
		}
		if f.triggerStatus != 0 {
			f.reply(w, f.triggerStatus, map[string]string{"detail": "Agent 'box' is not connected."})
			return
		}
		f.reply(w, 202, map[string]string{"message_id": "20000000-0000-4000-8000-000000000001"})
	default:
		f.reply(w, 405, map[string]string{"detail": "Method not allowed."})
	}
}

func TestControlHostBase(t *testing.T) {
	for in, want := range map[string]string{
		"spekk.example.com":          "https://spekk.example.com",
		"https://spekk.example.com/": "https://spekk.example.com",
		"localhost:8000":             "http://localhost:8000",
		"http://127.0.0.1:8000":      "http://127.0.0.1:8000",
		"http://[::1]:8000":          "http://[::1]:8000",
	} {
		got, err := controlHostBase(in)
		if err != nil || got != want {
			t.Errorf("controlHostBase(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// Plain http to a remote host would send the API token in clear text.
	for _, in := range []string{"http://spekk.example.com", "", "ftp://spekk.example.com"} {
		if got, err := controlHostBase(in); err == nil {
			t.Errorf("controlHostBase(%q) = %q, want an error", in, got)
		}
	}
}

func TestNewControlHostNamesMissingVariable(t *testing.T) {
	t.Setenv("SPEKK_HOST", "spekk.example.com")
	t.Setenv(apiTokenVar, "")
	if _, err := newControlHost(); err == nil || !strings.Contains(err.Error(), apiTokenVar) {
		t.Errorf("want an error naming %s, got %v", apiTokenVar, err)
	}
	t.Setenv(apiTokenVar, "tok")
	t.Setenv("SPEKK_HOST", "")
	if _, err := newControlHost(); err == nil || !strings.Contains(err.Error(), "SPEKK_HOST") {
		t.Errorf("want an error naming SPEKK_HOST, got %v", err)
	}
}

// A failed call names the call, the status, and the control host's reason,
// and never the token that authenticated it.
func TestControlHostErrorCarriesDetailNotToken(t *testing.T) {
	f := useControlHost(t)
	f.revokeStatus = http.StatusForbidden
	id := f.addAgent("box")
	c, err := newControlHost()
	if err != nil {
		t.Fatal(err)
	}
	err = c.revokeAgent(id)
	if err == nil {
		t.Fatal("a 403 must be an error")
	}
	for _, want := range []string{"POST", "/revoke/", "403", "revoke refused by the test"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
	if strings.Contains(err.Error(), testAPIToken) {
		t.Errorf("error %q carries the API token", err)
	}
	if got := f.requests[0].Header.Get("Authorization"); got != "Token "+testAPIToken {
		t.Errorf("Authorization = %q", got)
	}
}

// A redirect could send the API token to an address the user never chose,
// so the client does not follow one.
func TestControlHostDoesNotFollowRedirects(t *testing.T) {
	var reached bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true }))
	t.Cleanup(other.Close)
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(host.Close)
	t.Setenv("SPEKK_HOST", host.URL)
	t.Setenv(apiTokenVar, testAPIToken)

	c, err := newControlHost()
	if err != nil {
		t.Fatal(err)
	}
	err = c.revokeAgent("agent-1")
	if err == nil || !strings.Contains(err.Error(), "307") {
		t.Errorf("want an error with the 307 status, got %v", err)
	}
	if reached {
		t.Error("the client followed the redirect with the API token")
	}
}

// A user in several organizations must say which one the agent joins. The
// error says how, with the body the control host really sends.
func TestRegisterHintsAtOrganization(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		hint       bool
	}{
		{"several organizations", `{"organization": ["You belong to several organizations. Specify one."]}`, true},
		// The control host raises the error from its view, and today it
		// sends the text as a string, not a list.
		{"several organizations, string form", `{"organization": "You belong to several organizations. Specify one."}`, true},
		{"another field", `{"sandbox_id": ["agent with this sandbox id already exists."]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				io.WriteString(w, tc.body)
			}))
			t.Cleanup(host.Close)
			t.Setenv("SPEKK_HOST", host.URL)
			t.Setenv(apiTokenVar, testAPIToken)
			c, err := newControlHost()
			if err != nil {
				t.Fatal(err)
			}

			_, err = c.registerAgent("box", "")
			if err == nil {
				t.Fatal("a 400 must be an error")
			}
			if !strings.Contains(err.Error(), "400") {
				t.Errorf("error %q lacks the status", err)
			}
			for _, want := range []string{"--organization <id>", `"organization" field`, "GET /api/agents/"} {
				if got := strings.Contains(err.Error(), want); got != tc.hint {
					t.Errorf("hint %q present = %v, want %v:\n%s", want, got, tc.hint, err)
				}
			}
			if tc.hint && !strings.Contains(err.Error(), "You belong to several organizations") {
				t.Errorf("error %q drops the text of the control host", err)
			}
		})
	}
}

func TestRegistrationRejectsInvalidAgentID(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"bad-id","auth_token":"agent-token"}`)
	}))
	t.Cleanup(host.Close)
	t.Setenv("SPEKK_HOST", host.URL)
	t.Setenv(apiTokenVar, testAPIToken)
	c, err := newControlHost()
	if err != nil {
		t.Fatal(err)
	}
	if agent, err := c.registerAgent("box", ""); err == nil || agent != nil {
		t.Errorf("registerAgent accepted invalid ID: agent=%v, err=%v", agent, err)
	}
	if id, err := newRegistrationHost(host.URL, "registration-token").registerWithToken("box", "agent-token"); err == nil || id != "" {
		t.Errorf("registerWithToken accepted invalid ID: id=%q, err=%v", id, err)
	}
}
