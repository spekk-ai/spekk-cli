package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
)

func TestSandboxSessionReceivesAgentDeliveryVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	t.Setenv("SPEKK_AGENT_TOKEN", "made-up-token")
	dir := t.TempDir()
	script := "#!/bin/sh\n" + claudeVersionAnswer + `cat > /dev/null
if [ -n "$SPEKK_AGENT_TOKEN" ]; then token_state=present; else token_state=missing; fi
echo '{"type":"system","subtype":"init","session_id":"claude-session"}'
for i in $(seq 1 50); do
  [ -s "$SPEKK_CLAUDE_SESSION_FILE" ] && break
  sleep 0.1
done
resume_id=$(cat "$SPEKK_CLAUDE_SESSION_FILE")
echo "{\"type\":\"result\",\"session_id\":\"claude-session\",\"result\":\"$SPEKK_AGENT_SESSION_ID:$SPEKK_AGENT_TURN_ID:$token_state:$resume_id\"}"
`
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	if err := wsjson.Write(t.Context(), host.nextConn(t), Message{
		Type: MessageTypeMessage, Text: "Make a file", AgentSessionID: "agent-session-1", AgentTurnID: "agent-turn-1",
	}); err != nil {
		t.Fatal(err)
	}
	frame := host.awaitFrame(t, MessageTypeResult, 20*time.Second).frame
	if frame["output"] != "agent-session-1:agent-turn-1:present:claude-session" {
		t.Fatalf("session delivery variables = %v", frame["output"])
	}
}
