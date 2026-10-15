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

func toolStart(id string) string {
	return `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"` + id + `","name":"Bash"}]}}`
}

func toolEnd(id string) string {
	return `{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"` + id + `"}]}}`
}

func snapshot(tasks int) string {
	list := make([]string, tasks)
	for i := range list {
		list[i] = fmt.Sprintf(`{"task_id":"t%d"}`, i)
	}
	return `{"type":"system","subtype":"background_tasks_changed","tasks":[` + strings.Join(list, ",") + `]}`
}

const resultLine = `{"type":"result","subtype":"success","result":"done"}`

func observeAll(p *turnProgress, lines ...string) {
	for _, l := range lines {
		p.observe([]byte(l))
	}
}

func TestInFlightToolCalls(t *testing.T) {
	t.Run("one call in flight, then closed", func(t *testing.T) {
		p := newTurnProgress()
		if p.working() {
			t.Fatal("a new turn is working before any line")
		}
		observeAll(p, toolStart("a"))
		if !p.working() {
			t.Fatal("not working with a call in flight")
		}
		observeAll(p, toolEnd("a"))
		if p.working() {
			t.Fatal("working after the call ended")
		}
	})
	t.Run("two overlapping calls close in reverse order", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, toolStart("a"), toolStart("b"), toolEnd("b"))
		if !p.working() {
			t.Fatal("not working while call a is still in flight")
		}
		observeAll(p, toolEnd("a"))
		if p.working() {
			t.Fatal("working after both calls ended")
		}
	})
	t.Run("a result for an unknown id changes nothing", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, toolEnd("x"))
		if p.working() || len(p.inFlight) != 0 {
			t.Fatal("an unknown result changed the set")
		}
		observeAll(p, toolStart("a"), toolEnd("x"))
		if !p.working() {
			t.Fatal("an unknown result closed call a")
		}
	})
	t.Run("a call open at stream end does not reach the next turn", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, toolStart("a"))
		if next := newTurnProgress(); next.working() {
			t.Fatal("a new turn starts busy")
		}
	})
}

// The long foreground call is silent from its start to its end. The set must
// say "working" for every line of that span.
func TestFixtureLongToolCallIsInFlightAcrossItsSilentSpan(t *testing.T) {
	const long = "toolu_01H7qpvvQrT4qwC81ypoD4id"
	p := newTurnProgress()
	started, ended := false, false
	for _, line := range fixtureLines(t, "foreground-tool-calls.jsonl") {
		p.observe(line)
		_, open := p.inFlight[long]
		switch {
		case open:
			started = true
			if !p.working() {
				t.Fatal("not working while the long call is in flight")
			}
		case started:
			ended = true
		}
	}
	if !started || !ended {
		t.Fatalf("the fixture did not open and close the long call (started=%v ended=%v)", started, ended)
	}
	if p.working() {
		t.Fatal("working at the end of a stream where every call ended")
	}
}

func TestBackgroundTaskInventory(t *testing.T) {
	t.Run("no background task", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, toolStart("a"), toolEnd("a"), resultLine)
		if p.tasks != 0 || p.tasksAtResult != 0 {
			t.Fatalf("tasks = %d, at result = %d, want 0 and 0", p.tasks, p.tasksAtResult)
		}
	})
	t.Run("a task still live at stream end", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, snapshot(1), resultLine)
		if p.tasks != 1 || p.tasksAtResult != 1 {
			t.Fatalf("tasks = %d, at result = %d, want 1 and 1", p.tasks, p.tasksAtResult)
		}
	})
	t.Run("a task that finishes before stream end", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, snapshot(1), snapshot(0), resultLine)
		if p.tasks != 0 || p.tasksAtResult != 0 {
			t.Fatalf("tasks = %d, at result = %d, want 0 and 0", p.tasks, p.tasksAtResult)
		}
	})
	t.Run("the latest snapshot replaces the count", func(t *testing.T) {
		p := newTurnProgress()
		observeAll(p, snapshot(2), snapshot(0))
		if p.tasks != 0 {
			t.Fatalf("tasks = %d after an empty snapshot, want 0", p.tasks)
		}
		observeAll(p, snapshot(3))
		if p.tasks != 3 {
			t.Fatalf("tasks = %d, want 3", p.tasks)
		}
	})
}

// tasksAtEachResult replays a fixture and returns the inventory size at each
// result event, and at the end of the stream.
func tasksAtEachResult(t *testing.T, name string) (atResults []int, atEnd int) {
	t.Helper()
	p := newTurnProgress()
	for _, line := range fixtureLines(t, name) {
		p.observe(line)
		if decodeStreamLine(line).result {
			atResults = append(atResults, p.tasksAtResult)
		}
	}
	return atResults, p.tasks
}

func TestFixtureInventoryAtResults(t *testing.T) {
	if at, end := tasksAtEachResult(t, "background-task-outlives-turn.jsonl"); !reflect.DeepEqual(at, []int{1}) || end != 0 {
		t.Errorf("yield: tasks at results = %v, at end = %d; want [1] and 0 after the kill", at, end)
	}
	if at, end := tasksAtEachResult(t, "background-task-finishes-before-second-result.jsonl"); !reflect.DeepEqual(at, []int{1, 0}) || end != 0 {
		t.Errorf("completion: tasks at results = %v, at end = %d; want [1 0] and 0", at, end)
	}
}

func TestFixtureTerminal(t *testing.T) {
	cases := map[string]bool{
		"foreground-tool-calls.jsonl":                         true,
		"background-task-outlives-turn.jsonl":                 false,
		"background-task-finishes-before-second-result.jsonl": true,
	}
	for name, want := range cases {
		p := newTurnProgress()
		for _, line := range fixtureLines(t, name) {
			p.observe(line)
		}
		if got := p.terminal(); got != want {
			t.Errorf("%s: terminal = %v, want %v", name, got, want)
		}
	}
}
