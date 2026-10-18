package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
)

const (
	fakeSignInURL  = "https://claude.example/oauth?code_challenge=fake-challenge-7d1f"
	fakeSignInCode = "fake-code-93be41"
)

// fakeSignInClaude puts a claude on PATH that acts like the login: it prints
// a URL, reads one line, and exits 0 when the line is the code. It writes its
// arguments to a file, so the test sees what the agent ran.
func fakeSignInClaude(t *testing.T, script string) (argsFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")
	body := "#!/bin/sh\necho \"$@\" > '" + argsFile + "'\n" + script
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}

func startTerminalClient(t *testing.T) (*testControlHost, func(frame map[string]any)) {
	t.Helper()
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go client.Run(ctx)
	conn := host.nextConn(t)
	send := func(frame map[string]any) {
		if err := wsjson.Write(ctx, conn, frame); err != nil {
			t.Fatal(err)
		}
	}
	return host, send
}

// The login runs in a terminal, its bytes reach the control host, the code
// that the person types reaches the login, and the log holds neither.
func TestSignInTerminalRelaysAndRecordsNothing(t *testing.T) {
	logged := captureLog(t)
	argsFile := fakeSignInClaude(t, "echo 'Open "+fakeSignInURL+"'\nprintf 'Paste code: '\nread code\n[ \"$code\" = '"+fakeSignInCode+"' ] && exit 0\nexit 3\n")
	host, send := startTerminalClient(t)

	send(map[string]any{"type": "terminal_open", "terminal_id": "term-1", "cols": 100, "rows": 30, "agent_id": "agent-1", "user_id": "user-1"})
	var output strings.Builder
	for !strings.Contains(output.String(), "Paste code:") {
		got := host.awaitFrame(t, MessageTypeTerminalOutput, 10*time.Second)
		data, _ := base64.StdEncoding.DecodeString(got.frame["data"].(string))
		output.Write(data)
	}
	if !strings.Contains(output.String(), fakeSignInURL) {
		t.Errorf("terminal output %q has no sign-in URL", output.String())
	}
	send(map[string]any{"type": "terminal_input", "terminal_id": "term-1", "data": base64.StdEncoding.EncodeToString([]byte(fakeSignInCode + "\r"))})

	// The heartbeat comes before the exit, so the control host knows the new
	// sign-in state when the browser hears of the exit.
	var sawHeartbeat bool
	for {
		var got recvFrame
		select {
		case got = <-host.frames:
		case <-time.After(10 * time.Second):
			t.Fatal("no terminal_exit frame arrived")
		}
		if got.frame["type"] == MessageTypeHeartbeat {
			sawHeartbeat = true
		}
		if got.frame["type"] == MessageTypeTerminalExit {
			if !sawHeartbeat {
				t.Error("terminal_exit came before a heartbeat")
			}
			if got.frame["exit_code"] != float64(0) || got.frame["terminal_id"] != "term-1" {
				t.Errorf("exit frame = %v", got.frame)
			}
			break
		}
	}

	if args, _ := os.ReadFile(argsFile); strings.TrimSpace(string(args)) != "auth login" {
		t.Errorf("the agent ran claude %q, want %q", args, "auth login")
	}
	for _, secret := range []string{fakeSignInURL, fakeSignInCode} {
		if strings.Contains(logged.String(), secret) {
			t.Errorf("the log holds %q:\n%s", secret, logged)
		}
	}
}
