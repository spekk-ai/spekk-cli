package sandbox

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

// injectCodexAuth sends a ChatGPT login cache in the SSH script on stdin.
// A fresh private file on the sandbox is owned by agent. Codex updates this
// file in place when it refreshes its login.
func injectCodexAuth(ip, keyPath, name, user, path string) error {
	data, err := readCodexAuth(path)
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	script := "set -e\numask 077\ninstall -d -o agent -g agent -m 0700 /home/agent/.codex\n" +
		"if [ ! -s /home/agent/.codex/auth.json ]; then\n" +
		"f=$(mktemp /home/agent/.codex/auth.json.XXXXXX)\n" +
		"printf '%s' '" + encoded + "' | base64 -d > \"$f\"\n" +
		"chown agent:agent \"$f\"\nchmod 0600 \"$f\"\n" +
		"mv -f \"$f\" /home/agent/.codex/auth.json\nfi\n"
	out, err := runSecretScript(ip, keyPath, name, user, script)
	if err != nil {
		return fmt.Errorf("installing Codex auth file: %s\n%s", err, out)
	}
	return nil
}

func readCodexAuth(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading Codex auth file: %w", err)
	}
	var auth struct {
		AuthMode string `json:"auth_mode"`
		Tokens   struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(data, &auth); err != nil || auth.AuthMode != "chatgpt" || auth.Tokens.RefreshToken == "" {
		return nil, fmt.Errorf("Codex auth file must contain a ChatGPT login with a refresh token")
	}
	return data, nil
}
