package main

// streamEvent is what one event of claude's stream-json output says about the
// progress of a turn. The zero value means that nothing was observed.
//
// Every field comes from a JSON field of the event. Nothing here reads the
// agent's words (specs/turn-progress-reporting/).
type streamEvent struct {
	snapshot bool // the event is a background-task inventory snapshot
	tasks    int  // the size of that inventory, when snapshot is true
	result   bool // the event is a turn result
}

// decodeStreamEvent reads one decoded line of claude's stream-json output. It
// has no I/O and no state.
//
// These shapes are not a documented API of Claude Code, and the fixtures in
// testdata/ pin them. An event that does not match a known shape decodes to
// "nothing observed": an unknown type or subtype, a missing field, or a field
// of an unexpected JSON type. A future Claude Code release can then only make
// the client report less, never report a wrong outcome.
func decodeStreamEvent(event map[string]any) streamEvent {
	var ev streamEvent
	switch event["type"] {
	case "system":
		if tasks, ok := event["tasks"].([]any); ok && event["subtype"] == "background_tasks_changed" {
			ev.snapshot = true
			ev.tasks = len(tasks)
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
	// tasks is the size of the most recent background-task inventory. Each
	// snapshot replaces it. It is never counted up from starts and down from
	// ends, because one missed event in that scheme is a permanent error.
	tasks int
	// tasksAtResult is tasks at the most recent result event.
	tasksAtResult int
}

// observe updates the progress from one decoded line of claude's stream-json
// output.
func (p *turnProgress) observe(event map[string]any) {
	ev := decodeStreamEvent(event)
	if ev.snapshot {
		p.tasks = ev.tasks
	}
	if ev.result {
		p.tasksAtResult = p.tasks
	}
}

// terminal reports whether the turn ended the task, for the terminal field of
// the result frame. It is false when a background task was still in the
// inventory at the last result event, so the agent yielded while work it
// started was still live.
//
// The LAST result counts, not the first. A turn can emit more than one: Claude
// Code can wait for a background task after a result, and then emit a second
// result when the task ends. The first result of such a turn shows a live task
// for a turn that did complete.
//
// Only the inventory decides. The result text, subtype, stop_reason, and
// terminal_reason are never read: terminal_reason "completed" was seen on a
// turn that left a background task running.
func (p *turnProgress) terminal() bool {
	return p.tasksAtResult == 0
}
