package sandbox

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// newAgentToken decides the agent token for a sandbox that has no agent yet.
// With SPEKK_API_TOKEN set, it registers an agent and returns the token the
// control host made for it, and the agent's id. Without it, it makes a token
// itself and registers nothing, as create always did.
func newAgentToken(name, organization string) (token, agentID string, err error) {
	if strings.TrimSpace(os.Getenv(apiTokenVar)) == "" {
		return generateAgentToken(), "", nil
	}
	c, err := newControlHost()
	if err != nil {
		return "", "", err
	}
	fmt.Fprintln(os.Stderr, "Registering the agent on the control host...")
	agent, err := c.registerAgent(name, organization)
	if err != nil {
		return "", "", fmt.Errorf("registering the agent: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Registered agent %s.\n", agent.ID)
	return agent.AuthToken, agent.ID, nil
}

// revokeRecordedAgent revokes an agent by id with the user's API token.
func revokeRecordedAgent(id string) error {
	c, err := newControlHost()
	if err != nil {
		return err
	}
	return c.revokeAgent(id)
}

// revokeAbandoned revokes an agent that a failed command registered, says
// what happened, and reports whether the revoke succeeded. A failure names
// the id, because nothing else will.
func revokeAbandoned(id, name string) bool {
	if err := revokeRecordedAgent(id); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not revoke agent %s on the control host: %s\nRevoke it with: spekk sandbox destroy %s, or on the control host.\n", id, err, name)
		return false
	}
	fmt.Fprintf(os.Stderr, "Revoked agent %s on the control host, because the command failed.\n", id)
	return true
}

// Register registers a recorded sandbox on the control host: one created
// without SPEKK_API_TOKEN, or one whose agent must be replaced. It writes the
// new agent token to the machine and restarts the agent, then revokes the
// agent the record named before, whose token is no longer on the machine.
func Register(name, organization string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}
	if sandbox.IP == "" {
		return fmt.Errorf("sandbox %q has no IP address on record", name)
	}
	c, err := newControlHost()
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Registering the agent on the control host...")
	agent, err := c.registerAgent(name, organization)
	if err != nil {
		return fmt.Errorf("registering the agent: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Registered agent %s. Writing its token to %s...\n", agent.ID, machineRef(sandbox))

	user := sshUser(sandbox)
	if out, err := runSSHCombined(sandbox.IP, sandbox.SSHKeyPath, name, user, privilegedScript(user, buildTokenScript(agent.AuthToken))); err != nil {
		revokeAbandoned(agent.ID, name)
		return fmt.Errorf("writing the agent token to %s: %w\n%s", machineRef(sandbox), err, strings.TrimSpace(out))
	}

	previous := sandbox.AgentID
	sandbox.AgentID = agent.ID
	if err := SaveSandbox(name, sandbox); err != nil {
		// The machine runs the new agent, so revoking it would be wrong.
		return fmt.Errorf("saving metadata: %w (agent %s is registered and running on the machine)", err, agent.ID)
	}

	if previous != "" && previous != agent.ID {
		if err := c.revokeAgent(previous); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not revoke the previous agent %s: %s\nRevoke it on the control host.\n", previous, err)
		} else {
			fmt.Fprintf(os.Stderr, "Revoked the previous agent %s.\n", previous)
		}
	}
	fmt.Fprintf(os.Stderr, "Sandbox %q is registered as agent %s.\n", name, agent.ID)
	return nil
}

// buildTokenScript replaces the SPEKK_AGENT_TOKEN line of the agent's env
// file and restarts the agent. Every other line stays, because it holds the
// credentials that only the user's machine had. The token travels
// base64-encoded, so no character of it can change the script, and umask
// keeps the new file private before chmod runs.
func buildTokenScript(token string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(token))
	return fmt.Sprintf(`set -e
umask 077
f=/etc/spekk/agent.env
token=$(echo '%s' | base64 -d)
mkdir -p /etc/spekk
touch "$f"
grep -v '^SPEKK_AGENT_TOKEN=' "$f" > "$f.new" || true
printf 'SPEKK_AGENT_TOKEN=%%s\n' "$token" >> "$f.new"
chown agent:agent "$f.new"
chmod 600 "$f.new"
mv -f "$f.new" "$f"
systemctl restart spekk-agent`, encoded)
}

// readAgentToken is the seam Provision reads the machine's current agent
// token through. It reads stdout only, so an ssh warning on stderr cannot
// become part of the token. The last assignment wins, as it does for
// systemd.
var readAgentToken = func(meta *SandboxMeta, name string) (string, error) {
	user := sshUser(meta)
	out := runSSH(meta.IP, meta.SSHKeyPath, name, user, privilegedScript(user, `sed -n 's/^SPEKK_AGENT_TOKEN=//p' /etc/spekk/agent.env`))
	lines := strings.Fields(out)
	if len(lines) == 0 {
		return "", fmt.Errorf("no SPEKK_AGENT_TOKEN in /etc/spekk/agent.env, or the connection failed")
	}
	return lines[len(lines)-1], nil
}
