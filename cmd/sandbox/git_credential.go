package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// agentEnvFile is the env file of the agent service. The git credential
// helper reads its settings from this file and not from its environment:
// git runs the helper from each shell of agent, and only the service has
// the variables in its environment.
var agentEnvFile = "/etc/spekk/agent.env"

// runGitCredential is `agent-client git-credential <action>`, the git
// credential helper of a sandbox that a server launched. For a GitHub
// request, it gets a new installation token from the control host, so git
// keeps its access after the first token expires. It answers no other host
// and stores nothing: git keeps no token on the sandbox.
func runGitCredential(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "get" {
		return 0
	}
	req := readCredentialRequest(stdin)
	if req["protocol"] != "https" || req["host"] != "github.com" {
		return 0
	}
	token, err := fetchGitToken()
	if err != nil {
		fmt.Fprintf(stderr, "spekk git credential helper: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "username=x-access-token\npassword=%s\n", token)
	return 0
}

// readCredentialRequest reads the key=value lines that git writes to a
// helper, up to a blank line or the end.
func readCredentialRequest(r io.Reader) map[string]string {
	req := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			req[k] = v
		}
	}
	return req
}

// readEnvFile reads KEY=VALUE lines. The last value of a key wins, as it
// does for systemd.
func readEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(k, "#") {
			env[k] = v
		}
	}
	return env, nil
}

// fetchGitToken asks the control host for an installation token for the
// repository of this agent, with the agent token.
func fetchGitToken() (string, error) {
	env, err := readEnvFile(agentEnvFile)
	if err != nil {
		return "", err
	}
	host, agentID, agentToken := env["SPEKK_HOST"], env["SPEKK_AGENT_ID"], env["SPEKK_AGENT_TOKEN"]
	if host == "" || agentID == "" || agentToken == "" {
		return "", fmt.Errorf("%s has no SPEKK_HOST, SPEKK_AGENT_ID, or SPEKK_AGENT_TOKEN", agentEnvFile)
	}
	scheme := "https"
	if isLoopbackHost(host) {
		scheme = "http"
	}
	path := "/api/agents/" + url.PathEscape(agentID) + "/git-token/"
	req, err := http.NewRequest(http.MethodPost, scheme+"://"+host+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+agentToken)
	req.Header.Set("Accept", "application/json")
	// No redirect, so the agent token goes to the control host only.
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	var body struct {
		Token  string `json:"token"`
		Detail string `json:"detail"`
	}
	json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body)
	if resp.StatusCode != http.StatusOK || body.Token == "" {
		msg := fmt.Sprintf("POST %s: control host answered %d", path, resp.StatusCode)
		if body.Detail != "" {
			msg += ": " + body.Detail
		}
		return "", fmt.Errorf("%s", msg)
	}
	return body.Token, nil
}

// isLoopbackHost reports whether host, with or without a port, names this
// machine. Only such a host gets plain http.
func isLoopbackHost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
