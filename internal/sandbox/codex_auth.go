package sandbox

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	script := "set -e\numask 077\nd=/home/agent/.codex\na=$d/auth.json\nuid=$(id -u agent)\n" +
		"if [ -L \"$d\" ] || { [ -e \"$d\" ] && { [ ! -d \"$d\" ] || [ \"$(stat -c %u \"$d\")\" != \"$uid\" ]; }; }; then echo 'Unsafe Codex auth directory' >&2; exit 1; fi\n" +
		"if [ -L \"$a\" ] || { [ -e \"$a\" ] && { [ ! -f \"$a\" ] || [ \"$(stat -c %u \"$a\")\" != \"$uid\" ]; }; }; then echo 'Unsafe Codex auth file' >&2; exit 1; fi\n" +
		"install -d -o agent -g agent -m 0700 \"$d\"\n" +
		"f=$(mktemp /home/agent/.codex/auth.json.XXXXXX)\n" +
		"printf '%s' '" + encoded + "' | base64 -d > \"$f\"\n" +
		"chown agent:agent \"$f\"\nchmod 0600 \"$f\"\n" +
		"mv -f \"$f\" \"$a\"\n"
	out, err := runSecretScript(ip, keyPath, name, user, script)
	if err != nil {
		return fmt.Errorf("installing Codex auth file: %s\n%s", err, out)
	}
	return nil
}

func readCodexAuth(path string) ([]byte, error) {
	f, err := openCodexAuthFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
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
