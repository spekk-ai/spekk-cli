package docs

import "fmt"

// Problem is one source document that is not ok, with the command to run.
type Problem struct {
	State string
	ID    string
	Path  string
	Fix   string
}

// Problems lists every raw document that is not ok, then every file in raw/
// that has no stub, in that order.
func Problems(c *Corpus) []Problem {
	var out []Problem
	for _, d := range c.Docs {
		if d.Kind != KindRaw || d.State == StateOK {
			continue
		}
		p := Problem{State: d.State, ID: d.ID, Path: d.Path}
		switch d.State {
		case StateStale:
			p.Fix = fmt.Sprintf("the file changed since its stub was written: run spekk docs add %s, then write a new summary", d.Path)
		case StateNoSummary:
			p.Fix = fmt.Sprintf("read it (spekk docs show %s), then run spekk docs add %s --summary \"...\"", d.ID, d.Path)
		case StateMissingRaw:
			p.Fix = fmt.Sprintf("the file is gone: restore it, or delete %s", d.StubPath)
		case StatePointer:
			p.Fix = "the raw file is local to the clone that added it; search and show work from the text"
		}
		out = append(out, p)
	}
	for _, path := range c.NoStub {
		out = append(out, Problem{State: StateNoStub, Path: path, Fix: "run spekk docs add " + path})
	}
	return out
}

// StatusLines renders Problems one line each: state, id, path, fix.
func StatusLines(c *Corpus) []string {
	var out []string
	for _, p := range Problems(c) {
		id := p.ID
		if id == "" {
			id = "-"
		}
		out = append(out, fmt.Sprintf("%-11s  %s  %s  %s", p.State, id, p.Path, p.Fix))
	}
	return out
}
