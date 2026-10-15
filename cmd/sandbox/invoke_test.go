package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeFailingClaude puts an executable named claude on PATH that fails the way
// the real binary does when the model API is down: it gives up after its
// retries, writes the cause to a result event with is_error set, writes
// nothing useful to stderr, and exits 1.
func fakeFailingClaude(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
cat > /dev/null
echo '{"type":"system","subtype":"init","session_id":"sess-1"}'
echo '{"type":"system","subtype":"api_retry","attempt":10,"max_retries":10,"error_status":503,"error":"overloaded"}'
echo '{"type":"result","subtype":"success","is_error":true,"api_error_status":503,"session_id":"sess-1","result":"API Error: 503 Bedrock is unable to process your request."}'
exit 1
`
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestFailedTurnReportsTheAPIErrorItEndedOn is the regression test for a turn
// that ended on a model API outage and reported only "exit status 1". The
// cause was in the result event the whole time, and the client dropped it.
func TestFailedTurnReportsTheAPIErrorItEndedOn(t *testing.T) {
	fakeFailingClaude(t)
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	dispatchTurn(t, host.nextConn(t))
	got := host.awaitFrame(t, MessageTypeError, 20*time.Second)

	reason, _ := got.frame["error"].(string)
	if !strings.HasPrefix(reason, "claude exited: exit status 1") {
		t.Errorf("error = %q, want it to keep the exit status", reason)
	}
	if !strings.Contains(reason, "API Error: 503 Bedrock is unable to process your request.") {
		t.Errorf("error = %q, want the API error from the result event", reason)
	}
	if got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("error agent_session_id = %v, want agent-session-1", got.frame["agent_session_id"])
	}
	if _, ok := got.frame["terminal"]; ok {
		t.Errorf("error frame carries terminal = %v; only a result frame has it", got.frame["terminal"])
	}
}

// A non-error result is the turn's own output, not a cause, so a turn that
// exits non-zero after one reports the exit status alone.
func TestFailureReasonIgnoresANonErrorResult(t *testing.T) {
	got := failureReason(errors.New("exit status 1"), "here is the summary", false)
	if got != "claude exited: exit status 1" {
		t.Errorf("failureReason = %q, want the exit status alone", got)
	}
}

// TestStreamFramesNameTheirSession verifies that every stream frame carries
// the turn's agent_session_id, so the control host can attribute a line when
// several sessions stream at once.
func TestStreamFramesNameTheirSession(t *testing.T) {
	fakeFailingClaude(t)
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	dispatchTurn(t, host.nextConn(t))
	got := host.awaitFrame(t, MessageTypeStream, 20*time.Second)

	if got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("stream frame agent_session_id = %v, want agent-session-1", got.frame["agent_session_id"])
	}
}

// fakeReplayClaude puts an executable named claude on PATH that prints the
// stream in the file at path and exits 0.
func fakeReplayClaude(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	script := "#!/bin/sh\ncat > /dev/null\ncat '" + path + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestResultFrameDeclaresTerminal replays streams through the whole client.
// The field is always present and is a JSON boolean: the control host treats a
// string, a number, or null as absent.
func TestResultFrameDeclaresTerminal(t *testing.T) {
	// In the recorded runs, Claude Code waits for a background task after the
	// first result, and then emits a second result with an empty inventory.
	// That is true also when it stops the task at its time limit: the agent
	// sees the stop and reports it in the second result.
	cases := map[string]bool{
		"testdata/foreground-tool-calls.partial.jsonl":                         true,
		"testdata/background-task-stopped-at-time-limit.partial.jsonl":         true,
		"testdata/background-task-finishes-before-second-result.partial.jsonl": true,
	}

	// A first result while a task runs, then the task ends, then the process
	// exits with no second result. Claude Code 2.1.226 did this: it killed the
	// task a few seconds after the result. terminal must be false. The agent
	// gave its last word while the task was live, and no later result shows
	// that it saw how the task ended. The inventory does not say whether the
	// task finished or was killed, and in both cases the agent reported nothing
	// about it, so the turn yielded.
	yielded := filepath.Join(t.TempDir(), "yielded.jsonl")
	stream := strings.Join([]string{snapshot(1), resultLine, snapshot(0)}, "\n") + "\n"
	if err := os.WriteFile(yielded, []byte(stream), 0o644); err != nil {
		t.Fatal(err)
	}
	cases[yielded] = false

	for path, want := range cases {
		t.Run(filepath.Base(path), func(t *testing.T) {
			abs, err := filepath.Abs(path)
			if err != nil {
				t.Fatal(err)
			}
			fakeReplayClaude(t, abs)
			host := newTestControlHost(t)
			client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go client.Run(ctx)

			dispatchTurn(t, host.nextConn(t))
			got := host.awaitFrame(t, MessageTypeResult, 20*time.Second)

			terminal, ok := got.frame["terminal"].(bool)
			if !ok {
				t.Fatalf("terminal = %#v, want a JSON boolean", got.frame["terminal"])
			}
			if terminal != want {
				t.Errorf("terminal = %v, want %v", terminal, want)
			}
		})
	}
}
