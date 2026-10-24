package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
	"github.com/spekk-ai/spekk-cli/internal/claudecode"
)

// claudeVersionAnswer makes a fake claude answer the version check that the
// client runs before each turn, with a version above the minimum.
const claudeVersionAnswer = "[ \"$1\" = --version ] && { echo '2.1.301 (Claude Code)'; exit 0; }\n"

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
	script := "#!/bin/sh\n" + claudeVersionAnswer + `cat > /dev/null
echo '{"type":"system","subtype":"init","session_id":"sess-1"}'
echo '{"type":"system","subtype":"api_retry","attempt":10,"max_retries":10,"error_status":503,"error":"overloaded"}'
echo '{"type":"result","subtype":"success","is_error":true,"api_error_status":503,"session_id":"sess-1","result":"API Error: 503 Bedrock is unable to process your request.","modelUsage":{"m":{"inputTokens":1,"outputTokens":2,"cacheReadInputTokens":3,"cacheCreationInputTokens":4}},"total_cost_usd":0.01}'
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
	if _, ok := got.frame["usage"]; ok {
		t.Error("error frame carries usage")
	}
}

// fakeUsageClaude emits one recorded result per process. Its counter lets the
// same client receive two turns and exercise the cost baseline across workers.
func fakeUsageClaude(t *testing.T, results ...string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	for i, result := range results {
		path := filepath.Join(dir, fmt.Sprintf("result-%d", i+1))
		if err := os.WriteFile(path, []byte(result+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := "#!/bin/sh\n" + claudeVersionAnswer + fmt.Sprintf(`cat > /dev/null
counter=%q
n=0
[ -f "$counter" ] && n=$(cat "$counter")
n=$((n + 1))
echo "$n" > "$counter"
cat %q/result-"$n"
`, filepath.Join(dir, "counter"), dir)
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func usageResult(usage, cost string) string {
	return modelUsageResult(fmt.Sprintf(`{"claude-sonnet-5-5":%s}`, usage), cost)
}

func modelUsageResult(models, cost string) string {
	return fmt.Sprintf(`{"type":"result","session_id":"sess-1","result":"done","modelUsage":%s,"total_cost_usd":%s}`, models, cost)
}

const validUsage = `{"inputTokens":2,"outputTokens":4,"cacheReadInputTokens":6,"cacheCreationInputTokens":8}`

func TestResultFrameReportsProcessUsage(t *testing.T) {
	fakeUsageClaude(t, usageResult(validUsage, "0.0075909"), usageResult(`{"inputTokens":5,"outputTokens":10,"cacheReadInputTokens":15,"cacheCreationInputTokens":20}`, "0.0093454"))
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	conn := host.nextConn(t)
	for i, resumeID := range []string{"", "sess-1"} {
		if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Reply", SessionID: resumeID, AgentSessionID: "turn-1"}); err != nil {
			t.Fatal(err)
		}
		frame := host.awaitFrame(t, MessageTypeResult, 20*time.Second).frame
		for key, want := range map[string]any{"type": "result", "session_id": "sess-1", "agent_session_id": "turn-1", "output": "done", "terminal": true} {
			if frame[key] != want {
				t.Errorf("turn %d %s = %v, want %v", i+1, key, frame[key], want)
			}
		}
		usage, ok := frame["usage"].(map[string]any)
		if !ok {
			t.Fatalf("turn %d has no usage: %v", i+1, frame)
		}
		if len(usage) != 5 {
			t.Errorf("turn %d usage fields = %v", i+1, usage)
		}
		wantCounts := []map[string]float64{
			{"input_tokens": 2, "output_tokens": 4, "cache_read_input_tokens": 6, "cache_creation_input_tokens": 8},
			{"input_tokens": 3, "output_tokens": 6, "cache_read_input_tokens": 9, "cache_creation_input_tokens": 12},
		}[i]
		for key, want := range wantCounts {
			if usage[key] != want {
				t.Errorf("turn %d %s = %v, want %v", i+1, key, usage[key], want)
			}
		}
		wantCost := []float64{0.0075909, 0.0017545}[i]
		if got, ok := usage["cost_usd"].(float64); !ok || got < wantCost-0.00000001 || got > wantCost+0.00000001 {
			t.Errorf("turn %d cost = %v, want %v", i+1, usage["cost_usd"], wantCost)
		}
	}
}

func TestResultFrameSumsBackgroundTaskUsage(t *testing.T) {
	cases := []struct {
		file          string
		result        string
		input         float64
		output        float64
		cacheRead     float64
		cacheCreation float64
		costUSD       float64
	}{
		{"background-task-finishes-before-second-result.jsonl", "", 8, 118, 45332, 6531, 0.0318532},
		{"background-task-stopped-at-time-limit.jsonl", "", 8, 318, 45343, 6610, 0.0341703},
		{"two models", modelUsageResult(`{"claude-opus-4-1":{"inputTokens":2,"outputTokens":4,"cacheReadInputTokens":6,"cacheCreationInputTokens":8},"claude-haiku-4-5":{"inputTokens":3,"outputTokens":5,"cacheReadInputTokens":7,"cacheCreationInputTokens":9}}`, "0.02"), 5, 9, 13, 17, 0.02},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			if tc.result != "" {
				fakeUsageClaude(t, tc.result)
			} else {
				path, err := filepath.Abs(filepath.Join("testdata", tc.file))
				if err != nil {
					t.Fatal(err)
				}
				fakeReplayClaude(t, path)
			}
			host := newTestControlHost(t)
			client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go client.Run(ctx)

			dispatchTurn(t, host.nextConn(t))
			frame := host.awaitFrame(t, MessageTypeResult, 20*time.Second).frame
			usage, ok := frame["usage"].(map[string]any)
			if !ok {
				t.Fatalf("result has no usage: %v", frame)
			}
			for key, want := range map[string]float64{
				"input_tokens":                tc.input,
				"output_tokens":               tc.output,
				"cache_read_input_tokens":     tc.cacheRead,
				"cache_creation_input_tokens": tc.cacheCreation,
			} {
				if usage[key] != want {
					t.Errorf("%s = %v, want %v", key, usage[key], want)
				}
			}
			if usage["cost_usd"] != tc.costUSD {
				t.Errorf("cost_usd = %v, want %v", usage["cost_usd"], tc.costUSD)
			}
		})
	}
}

func TestResultFrameOmitsUnmeasurableUsage(t *testing.T) {
	cases := []struct {
		name     string
		result   string
		resumeID string
	}{
		{"old Claude", `{"type":"result","session_id":"sess-1","result":"done"}`, ""},
		{"empty modelUsage", modelUsageResult("{}", "0.01"), ""},
		{"unknown baseline", usageResult(validUsage, "0.01"), "sess-1"},
		{"missing count", usageResult(`{"inputTokens":1,"outputTokens":2,"cacheReadInputTokens":3}`, "0.01"), ""},
		{"fractional count", usageResult(`{"inputTokens":1.0,"outputTokens":2,"cacheReadInputTokens":3,"cacheCreationInputTokens":4}`, "0.01"), ""},
		{"count over limit", usageResult(`{"inputTokens":10000000001,"outputTokens":2,"cacheReadInputTokens":3,"cacheCreationInputTokens":4}`, "0.01"), ""},
		{"negative count", usageResult(`{"inputTokens":-1,"outputTokens":2,"cacheReadInputTokens":3,"cacheCreationInputTokens":4}`, "0.01"), ""},
		{"missing cost", fmt.Sprintf(`{"type":"result","session_id":"sess-1","result":"done","modelUsage":{"claude-sonnet-5-5":%s}}`, validUsage), ""},
		{"cost over limit", usageResult(validUsage, "1001"), ""},
		{"nonfinite cost", usageResult(validUsage, "1e999"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeUsageClaude(t, tc.result)
			host := newTestControlHost(t)
			client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go client.Run(ctx)
			if err := wsjson.Write(ctx, host.nextConn(t), Message{Type: MessageTypeMessage, Text: "Reply", SessionID: tc.resumeID, AgentSessionID: "turn-1"}); err != nil {
				t.Fatal(err)
			}
			frame := host.awaitFrame(t, MessageTypeResult, 20*time.Second).frame
			if _, ok := frame["usage"]; ok {
				t.Errorf("result carried unmeasurable usage: %v", frame["usage"])
			}
			if frame["output"] != "done" || frame["terminal"] != true {
				t.Errorf("result did not end the turn as before: %v", frame)
			}
		})
	}
}

func TestUsageTotalFallsBackAfterNegativeDifference(t *testing.T) {
	var totals usageTotals
	decode := func(total string) map[string]any {
		var event map[string]any
		decoder := json.NewDecoder(strings.NewReader(usageResult(validUsage, total)))
		decoder.UseNumber()
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		return event
	}
	first, ok := totals.measure("sess-1", "", decode("2"))
	if !ok || first.CostUSD != 2 {
		t.Fatalf("first cost = %v, %v", first.CostUSD, ok)
	}
	if _, ok := totals.measure("sess-1", "sess-1", decode("1")); ok {
		t.Fatal("negative difference reported usage")
	}
	third, ok := totals.measure("sess-1", "sess-1", decode("1.5"))
	if !ok || third.CostUSD != 0.5 {
		t.Errorf("cost after reset = %v, %v, want 0.5", third.CostUSD, ok)
	}
	lowerTokens := decode("1.75")
	lowerTokens["modelUsage"].(map[string]any)["claude-sonnet-5-5"].(map[string]any)["inputTokens"] = json.Number("1")
	if _, ok := totals.measure("sess-1", "sess-1", lowerTokens); ok {
		t.Fatal("negative token difference reported usage")
	}
	noTokens := decode("2")
	noTokens["modelUsage"] = nil
	if _, ok := totals.measure("sess-1", "sess-1", noTokens); ok {
		t.Fatal("missing tokens reported usage")
	}
	last, ok := totals.measure("sess-1", "sess-1", decode("2.25"))
	if ok {
		t.Errorf("usage after missing token baseline = %v, want none", last)
	}
	recovered, ok := totals.measure("sess-1", "sess-1", decode("2.5"))
	if !ok || recovered.CostUSD != 0.25 {
		t.Errorf("cost after restored token baseline = %v, %v, want 0.25", recovered.CostUSD, ok)
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
	script := "#!/bin/sh\n" + claudeVersionAnswer + "cat > /dev/null\ncat '" + path + "'\n"
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

// A turn on a Claude Code older than the minimum is refused before claude
// starts. 2.1.197 kills a background task seconds after the result, and the
// person sees a failed turn with no cause. The refusal names the version, the
// minimum and the remedy, all in the error field, because the control host
// reads no other field of the frame.
func TestTurnRefusedOnOldClaudeCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	ran := filepath.Join(dir, "ran")
	script := "#!/bin/sh\n[ \"$1\" = --version ] && { echo '2.1.197 (Claude Code)'; exit 0; }\ntouch '" + ran + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	dispatchTurn(t, host.nextConn(t))
	got := host.awaitFrame(t, MessageTypeError, 20*time.Second)

	want := "Claude Code 2.1.197 at " + filepath.Join(dir, "claude") + " is older than " + claudecode.MinVersion + ", the minimum for this sandbox. Run `spekk sandbox deploy` for this sandbox to install a current version."
	if got.frame["error"] != want {
		t.Errorf("error = %q, want %q", got.frame["error"], want)
	}
	if got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("error agent_session_id = %v, want agent-session-1", got.frame["agent_session_id"])
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("claude ran a turn on a version older than the minimum")
	}
}

// A turn runs when the client cannot read the Claude Code version. A later
// Claude Code can print its version in a form that the parse does not know,
// and a refusal would then stop every turn on every sandbox.
func TestTurnRunsOnUnreadableClaudeCodeVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
[ "$1" = --version ] && { echo 'Claude Code (next build)'; exit 0; }
cat > /dev/null
echo '{"type":"system","subtype":"init","session_id":"sess-1"}'
echo '{"type":"result","subtype":"success","is_error":false,"session_id":"sess-1","result":"done"}'
`
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	dispatchTurn(t, host.nextConn(t))
	got := host.awaitFrame(t, MessageTypeResult, 20*time.Second)
	if got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("result agent_session_id = %v, want agent-session-1", got.frame["agent_session_id"])
	}
}
