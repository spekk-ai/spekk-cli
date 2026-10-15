package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
	for _, id := range ev.toolStarts {
		out = append(out, "tool_start "+id)
	}
	for _, id := range ev.toolEnds {
		out = append(out, "tool_end "+id)
	}
	if ev.snapshot {
		out = append(out, fmt.Sprintf("tasks %d", ev.tasks))
	}
	if ev.result {
		out = append(out, "result")
	}
	return out
}

func TestFixturesReplayToTheExpectedEvents(t *testing.T) {
	cases := map[string][]string{
		"foreground-tool-calls.jsonl": {
			"tool_start toolu_01H7qpvvQrT4qwC81ypoD4id",
			"tool_start toolu_01466wk7F5EizqFT7xFVSKWF",
			"tool_start toolu_01CYe6cwnHQQkJF7no75VdXn",
			"tool_end toolu_01CYe6cwnHQQkJF7no75VdXn",
			"tool_end toolu_01466wk7F5EizqFT7xFVSKWF",
			"tool_end toolu_01H7qpvvQrT4qwC81ypoD4id",
			"result",
		},
		"background-task-outlives-turn.jsonl": {
			"tool_start toolu_01DHv4sVQt9m6AAWhWLofnPs",
			"tasks 1",
			"tool_end toolu_01DHv4sVQt9m6AAWhWLofnPs",
			"result",
			"tasks 0",
		},
		"background-task-finishes-before-second-result.jsonl": {
			"tool_start toolu_0149z7ZkEVJuDBMTZ4thfL3h",
			"tasks 1",
			"tool_end toolu_0149z7ZkEVJuDBMTZ4thfL3h",
			"result",
			"tasks 0",
			"result",
		},
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			var got []string
			for _, line := range fixtureLines(t, name) {
				got = append(got, describe(decodeStreamLine(line))...)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("decoded events changed, so a Claude Code stream shape changed\n got: %q\nwant: %q", got, want)
			}
		})
	}
}

// Each fixture starts with the init event of the run, which records the Claude
// Code version that the shapes were captured from.
func TestFixturesRecordTheirClaudeCodeVersion(t *testing.T) {
	for _, name := range fixtureNames(t) {
		var init struct {
			Subtype string `json:"subtype"`
			Version string `json:"claude_code_version"`
		}
		if err := json.Unmarshal(fixtureLines(t, name)[0], &init); err != nil || init.Subtype != "init" || init.Version == "" {
			t.Errorf("%s: the first line is not an init event with claude_code_version", name)
		}
	}
}

// Tool calls are content blocks inside assistant and user events. A real
// stream has no top-level tool_use or tool_result event, which is why the
// decoder recognizes none.
func TestFixturesHaveNoTopLevelToolEvents(t *testing.T) {
	for _, name := range fixtureNames(t) {
		for i, line := range fixtureLines(t, name) {
			var ev struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(line, &ev); err != nil {
				t.Fatalf("%s line %d is not JSON: %v", name, i+1, err)
			}
			if ev.Type == "tool_use" || ev.Type == "tool_result" {
				t.Errorf("%s line %d has a top-level %q event", name, i+1, ev.Type)
			}
		}
	}
}

func fixtureNames(t *testing.T) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join("testdata", "*.jsonl"))
	if err != nil || len(names) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for i, n := range names {
		names[i] = filepath.Base(n)
	}
	return names
}

// Input that matches no known shape observes nothing. It never panics and
// never produces a verdict.
func TestUnrecognizedLinesObserveNothing(t *testing.T) {
	lines := map[string]string{
		"not JSON":                   `{"type":"assistant"`,
		"empty":                      ``,
		"JSON array":                 `[1,2]`,
		"unknown type":               `{"type":"stream_event","event":{"type":"content_block_start"}}`,
		"unknown system subtype":     `{"type":"system","subtype":"task_started","task_id":"t1"}`,
		"top-level tool_use":         `{"type":"tool_use","id":"toolu_1"}`,
		"top-level tool_result":      `{"type":"tool_result","tool_use_id":"toolu_1"}`,
		"assistant with no message":  `{"type":"assistant"}`,
		"tool_use with no id":        `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash"}]}}`,
		"tool_result with no id":     `{"type":"user","message":{"content":[{"type":"tool_result","content":"ok"}]}}`,
		"tool_result in assistant":   `{"type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1"}]}}`,
		"user content as a string":   `{"type":"user","message":{"content":"hello"}}`,
		"id of the wrong type":       `{"type":"assistant","message":{"content":[{"type":"tool_use","id":7}]}}`,
		"snapshot with no tasks":     `{"type":"system","subtype":"background_tasks_changed"}`,
		"snapshot with null tasks":   `{"type":"system","subtype":"background_tasks_changed","tasks":null}`,
		"snapshot with string tasks": `{"type":"system","subtype":"background_tasks_changed","tasks":"t1"}`,
	}
	for name, line := range lines {
		if got := decodeStreamLine([]byte(line)); !reflect.DeepEqual(got, streamEvent{}) {
			t.Errorf("%s: decoded %+v, want nothing observed", name, got)
		}
	}
}
