package sandbox

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// apiTokenVar holds the user's spekk API token. It is the only credential of
// the user that a request to the control host carries. Cloud and model
// credentials go to the sandbox only; see specs/sandbox-self-register/.
const apiTokenVar = "SPEKK_API_TOKEN"

// controlHost calls the control host API as the user who owns the API token,
// or, in machine mode, with a one-time registration token.
type controlHost struct {
	base string // scheme and host, no trailing slash
	auth string // the whole Authorization header value
	http *http.Client
}

// newControlHost reads SPEKK_HOST and SPEKK_API_TOKEN, and refuses an address
// that would send the token in clear text.
func newControlHost() (*controlHost, error) {
	token := strings.TrimSpace(os.Getenv(apiTokenVar))
	if token == "" {
		return nil, fmt.Errorf("%s is not set: create an API token on the control host and export it", apiTokenVar)
	}
	base, err := controlHostBase(os.Getenv("SPEKK_HOST"))
	if err != nil {
		return nil, err
	}
	return &controlHost{base: base, auth: "Token " + token, http: noRedirectClient()}, nil
}

// newRegistrationHost calls the control host at base with a one-time
// registration token, which is a Bearer token and not a user's API token.
func newRegistrationHost(base, registrationToken string) *controlHost {
	return &controlHost{base: base, auth: "Bearer " + registrationToken, http: noRedirectClient()}
}

// noRedirectClient does not follow a redirect, so the Authorization header
// goes to the control host only. The 3xx answer becomes an error like any
// other.
func noRedirectClient() *http.Client {
	return &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// controlHostBase turns SPEKK_HOST into a base URL. A bare host gets https,
// except a loopback host, which gets http as the agent client does. Plain
// http to any other host is refused, because the API token would cross the
// network in clear text.
func controlHostBase(host string) (string, error) {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		return "", errors.New("SPEKK_HOST is not set: export the address of the control host")
	}
	if !strings.Contains(host, "://") {
		scheme := "https"
		if isLoopback(host) {
			scheme = "http"
		}
		host = scheme + "://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid SPEKK_HOST %q", host)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopback(u.Host) {
			return "", fmt.Errorf("SPEKK_HOST %q uses http: the API token would cross the network in clear text. Use https", host)
		}
	default:
		return "", fmt.Errorf("invalid SPEKK_HOST %q: use https://<host>", host)
	}
	return u.Scheme + "://" + u.Host, nil
}

// isLoopback reports whether hostport names this machine.
func isLoopback(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// authorize is the one place that puts a credential on a control host
// request, so a change to the token scheme changes this line only.
func (c *controlHost) authorize(req *http.Request) {
	req.Header.Set("Authorization", c.auth)
}

// do sends one request and decodes a 2xx body into out, when out is not nil.
// The error for any other status names the call and the status, and the
// control host's "detail" text when it has one. It never quotes the request,
// so it cannot carry the token.
func (c *controlHost) do(method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	c.authorize(req)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &apiError{Method: method, Path: path, Status: resp.StatusCode, Detail: errorDetail(data), body: data}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s %s: reading the response: %w", method, path, err)
	}
	return nil
}

// apiError is a response from the control host that is not 2xx.
type apiError struct {
	Method, Path string
	Status       int
	Detail       string
	body         []byte
}

// hasField reports whether the error body is a JSON object with the key
// name, as a validation error for that field is.
func (e *apiError) hasField(name string) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(e.body, &fields) != nil {
		return false
	}
	_, ok := fields[name]
	return ok
}

func (e *apiError) Error() string {
	msg := fmt.Sprintf("%s %s: control host answered %d %s", e.Method, e.Path, e.Status, http.StatusText(e.Status))
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

// errorDetail reads the reason from an error body: the "detail" field, or the
// field errors of a validation failure. A body that is not JSON gives "", so
// an HTML error page does not flood the terminal.
func errorDetail(body []byte) string {
	var fields map[string]any
	if json.Unmarshal(body, &fields) != nil {
		return ""
	}
	if d, ok := fields["detail"].(string); ok {
		return d
	}
	compact, err := json.Marshal(fields)
	if err != nil || len(fields) == 0 {
		return ""
	}
	return string(compact)
}

// registeredAgent is the part of the register response the CLI uses.
// AuthToken is shown one time only. It goes to the machine and nowhere else.
type registeredAgent struct {
	ID        string `json:"id"`
	AuthToken string `json:"auth_token"`
}

// registerAgent creates an agent for the caller. sandbox_id must be unique on
// the control host, and a revoked agent keeps its own, so each registration
// sends a new one: a sandbox destroyed and created again under the same name
// can then register again.
func (c *controlHost) registerAgent(name, organization string) (*registeredAgent, error) {
	suffix := make([]byte, 4)
	rand.Read(suffix)
	body := map[string]string{
		"name":       name,
		"sandbox_id": "spekk-" + name + "-" + hex.EncodeToString(suffix),
	}
	if organization != "" {
		body["organization"] = organization
	}
	var agent registeredAgent
	if err := c.do(http.MethodPost, "/api/agents/", body, &agent); err != nil {
		// The control host could not choose the organization: the user
		// belongs to several, or is not a member of the one given.
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest && apiErr.hasField("organization") {
			return nil, fmt.Errorf("%w\nPass --organization <id> to choose the organization. The id is in the \"organization\" field of an agent you already have (GET /api/agents/ on the control host), or ask an admin of the organization", err)
		}
		return nil, err
	}
	if !uuidRe.MatchString(agent.ID) {
		return nil, errors.New("POST /api/agents/: the response has an invalid agent id")
	}
	if agent.AuthToken == "" {
		return nil, errors.New("POST /api/agents/: the response has no agent token")
	}
	return &agent, nil
}

// revokeAgent stops the agent from authenticating, and closes its connection.
func (c *controlHost) revokeAgent(id string) error {
	return c.do(http.MethodPost, "/api/agents/"+url.PathEscape(id)+"/revoke/", nil, nil)
}

// registerWithToken registers the agent of a machine-mode sandbox. The
// caller made the agent token, and the control host stores its hash, so the
// raw token exists only in this process and on the machine. sandbox_id is
// new for each registration, as in registerAgent.
func (c *controlHost) registerWithToken(name, agentToken string) (string, error) {
	suffix := make([]byte, 4)
	rand.Read(suffix)
	body := map[string]string{
		"name":       name,
		"sandbox_id": "spekk-" + name + "-" + hex.EncodeToString(suffix),
		"token":      agentToken,
	}
	var agent struct {
		ID string `json:"id"`
	}
	if err := c.do(http.MethodPost, "/api/agents/register/", body, &agent); err != nil {
		return "", err
	}
	if !uuidRe.MatchString(agent.ID) {
		return "", errors.New("POST /api/agents/register/: the response has an invalid agent id")
	}
	return agent.ID, nil
}
