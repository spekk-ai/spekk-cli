package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fixtureLines reads a captured stream from testdata/, one line per element.
func fixtureLines(t *testing.T, name string) [][]byte {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, append([]byte(nil), scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return lines
}

// describe names what a decoded line observed, so a replay can compare one
// readable sequence and a failure names the event that changed.
func describe(ev streamEvent) []string {
	var out []string
	if ev.snapshot {
		out = append(out, fmt.Sprintf("tasks %d", ev.tasks))
	}
	if ev.result {
		out = append(out, "result")
	}
	return out
}

// decodeLine decodes a line as invoke does: a line that is not a JSON object
// observes nothing.
func decodeLine(line []byte) streamEvent {
	var event map[string]any
	if json.Unmarshal(line, &event) != nil {
		return streamEvent{}
	}
	return decodeStreamEvent(event)
}

// Each scenario was recorded twice: once with the flags of the README, and
// once with --include-partial-messages, which the client passes to claude.
// Both must decode to the same events.
func TestFixturesReplayToTheExpectedEvents(t *testing.T) {
	twoResults := []string{"tasks 1", "result", "tasks 0", "result"}
	cases := map[string][]string{
		"foreground-tool-calls":                         {"result"},
		"background-task-stopped-at-time-limit":         twoResults,
		"background-task-finishes-before-second-result": twoResults,
	}
	for scenario, want := range cases {
		for _, name := range []string{scenario + ".jsonl", scenario + ".partial.jsonl"} {
			t.Run(name, func(t *testing.T) {
				var got []string
				for _, line := range fixtureLines(t, name) {
					got = append(got, describe(decodeLine(line))...)
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("decoded events changed, so a Claude Code stream shape changed\n got: %q\nwant: %q", got, want)
				}
			})
		}
	}
}

// Input that matches no known shape observes nothing. It never panics and
// never produces a verdict.
func TestUnrecognizedLinesObserveNothing(t *testing.T) {
	lines := map[string]string{
		"not JSON":                    `{"type":"system"`,
		"empty":                       ``,
		"JSON array":                  `[1,2]`,
		"unknown type":                `{"type":"stream_event","event":{"type":"content_block_start"}}`,
		"unknown system subtype":      `{"type":"system","subtype":"task_started","task_id":"t1"}`,
		"type of the wrong JSON type": `{"type":7,"subtype":"background_tasks_changed","tasks":[]}`,
		"snapshot with no tasks":      `{"type":"system","subtype":"background_tasks_changed"}`,
		"snapshot with null tasks":    `{"type":"system","subtype":"background_tasks_changed","tasks":null}`,
		"snapshot with string tasks":  `{"type":"system","subtype":"background_tasks_changed","tasks":"t1"}`,
	}
	for name, line := range lines {
		if got := decodeLine([]byte(line)); got != (streamEvent{}) {
			t.Errorf("%s: decoded %+v, want nothing observed", name, got)
		}
	}
}

func snapshot(tasks int) string {
	list := make([]string, tasks)
	for i := range list {
		list[i] = fmt.Sprintf(`{"task_id":"t%d"}`, i)
	}
	return `{"type":"system","subtype":"background_tasks_changed","tasks":[` + strings.Join(list, ",") + `]}`
}

const resultLine = `{"type":"result","subtype":"success","result":"done"}`

// observeAll feeds each line to p. The lines are constants of the tests, so a
// line that is not JSON is a bug in the test.
func observeAll(p *turnProgress, lines ...string) {
	for _, l := range lines {
		var event map[string]any
		if err := json.Unmarshal([]byte(l), &event); err != nil {
			panic(err)
		}
		p.observe(event)
	}
}

// The inventory is a full replacement, so a missed snapshot corrects itself on
// the next one.
func TestLatestSnapshotReplacesTheCount(t *testing.T) {
	var p turnProgress
	observeAll(&p, snapshot(2), snapshot(0))
	if p.tasks != 0 {
		t.Fatalf("tasks = %d after an empty snapshot, want 0", p.tasks)
	}
	observeAll(&p, snapshot(3))
	if p.tasks != 3 {
		t.Fatalf("tasks = %d, want 3", p.tasks)
	}
}
