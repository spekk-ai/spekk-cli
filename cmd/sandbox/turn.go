package main

import "encoding/json"

// streamEvent is what one line of claude's stream-json output says about the
// progress of a turn. The zero value means that nothing was observed.
//
// Every field comes from a JSON field of the line. Nothing here reads the
// agent's words (specs/turn-progress-reporting/).
type streamEvent struct {
	toolStarts []string // ids of the tool calls that started
	toolEnds   []string // tool_use_ids of the tool calls that ended
	snapshot   bool     // the line is a background-task inventory snapshot
	tasks      int      // the size of that inventory, when snapshot is true
	result     bool     // the line is a turn result
}

// wireEvent is the part of a stream-json event that decodeStreamLine reads.
//
// Tool calls are content blocks nested in assistant and user events. Claude
// Code emits no top-level tool_use or tool_result event, so none is decoded
// here. internal/serve/serve.go switches on those top-level types, and those
// branches never run.
type wireEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message *struct {
		Content []struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			ToolUseID string `json:"tool_use_id"`
		} `json:"content"`
	} `json:"message"`
	Tasks *[]json.RawMessage `json:"tasks"`
}

// decodeStreamLine reads one line of claude's stream-json output. It has no
// I/O and no state.
//
// These shapes are not a documented API of Claude Code, and the fixtures in
// testdata/ pin them. A line that does not match a known shape decodes to
// "nothing observed": not valid JSON, an unknown type or subtype, a missing
// field, or a field of an unexpected JSON type. A future Claude Code release
// can then only make the client report less, never report a wrong outcome.
func decodeStreamLine(line []byte) streamEvent {
	var ev streamEvent
	var w wireEvent
	if json.Unmarshal(line, &w) != nil {
		return ev
	}
	switch w.Type {
	case "assistant", "user":
		if w.Message == nil {
			return ev
		}
		for _, block := range w.Message.Content {
			switch {
			case w.Type == "assistant" && block.Type == "tool_use" && block.ID != "":
				ev.toolStarts = append(ev.toolStarts, block.ID)
			case w.Type == "user" && block.Type == "tool_result" && block.ToolUseID != "":
				ev.toolEnds = append(ev.toolEnds, block.ToolUseID)
			}
		}
	case "system":
		if w.Subtype == "background_tasks_changed" && w.Tasks != nil {
			ev.snapshot = true
			ev.tasks = len(*w.Tasks)
		}
	case "result":
		ev.result = true
	}
	return ev
}

// turnProgress is what the client has seen of the work of one invocation. A
// new one starts for each invocation and is dropped when the stream ends, so
// nothing from one turn can carry into the next.
type turnProgress struct {
	// inFlight holds the ids of the tool calls that started and did not end.
	// A start and an end pair by id only, because several calls can be in
	// flight at a time and their results can return in any order.
	inFlight map[string]struct{}
	// tasks is the size of the most recent background-task inventory. Each
	// snapshot replaces it. It is never counted up from starts and down from
	// ends, because one missed event in that scheme is a permanent error.
	tasks int
	// tasksAtResult is tasks at the most recent result event.
	tasksAtResult int
}

func newTurnProgress() *turnProgress {
	return &turnProgress{inFlight: map[string]struct{}{}}
}

// observe updates the progress from one line of claude's stream-json output.
func (p *turnProgress) observe(line []byte) {
	ev := decodeStreamLine(line)
	for _, id := range ev.toolStarts {
		p.inFlight[id] = struct{}{}
	}
	for _, id := range ev.toolEnds {
		delete(p.inFlight, id)
	}
	if ev.snapshot {
		p.tasks = ev.tasks
	}
	if ev.result {
		p.tasksAtResult = p.tasks
	}
}

// working reports whether a tool call is in flight. It is read from the set
// each time, not stored, so a missed event cannot leave it stale.
func (p *turnProgress) working() bool {
	return len(p.inFlight) > 0
}
