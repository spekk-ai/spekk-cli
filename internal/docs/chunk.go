// Package docs reads the document corpus that the index searches: the source
// documents in raw/ with their stubs in formatted/, and the prose that already
// lives in the repository (docs/, READMEs). It finds the documents, cuts their
// plaintext into passages, converts other formats to plaintext, and writes
// stubs. It holds no database code: internal/index writes what this package
// reads into .spekk/index.db.
package docs

import (
	"strings"
)

// Passage is one searchable part of a document. Start and End are 1-based,
// inclusive line numbers in the file as it is on disk, so `spekk docs show
// --lines Start:End` prints exactly this passage.
type Passage struct {
	Heading string
	Start   int
	End     int
	Text    string
}

// A section longer than maxSectionLines is cut into windows of windowLines.
// BM25 over a long section ranks badly, and the agent then reads the whole
// section to find one fact.
const (
	maxSectionLines = 80
	windowLines     = 40
)

// section is a heading section before the window rule: lines [start, end) as
// 0-based indexes into the file's lines.
type section struct {
	heading    string
	start, end int
}

// Chunk cuts the plaintext of a document into passages. A passage starts at
// each ATX or setext heading outside a fenced code block. A leading YAML
// frontmatter block is in no passage, but its lines are counted, so the line
// numbers are those of the file.
func Chunk(text string) []Passage {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	first := frontmatterEnd(lines)
	var sections []section
	cur := section{start: first}
	fence := ""
	for i := first; i < len(lines); i++ {
		line := lines[i]
		if f := fenceMarker(line); f != "" {
			switch {
			case fence == "":
				fence = f
			case strings.HasPrefix(strings.TrimSpace(line), fence):
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if h, ok := atxHeading(line); ok {
			cur.end = i
			sections = append(sections, cur)
			cur = section{heading: h, start: i}
			continue
		}
		// A setext underline makes the line above it the heading. The line
		// above must be a non-blank text line of the current section, and not
		// the heading line of an ATX section.
		if i > cur.start && isSetextUnderline(line) && strings.TrimSpace(lines[i-1]) != "" {
			if _, isATX := atxHeading(lines[i-1]); !isATX && !(i-1 == cur.start && cur.heading != "") {
				cur.end = i - 1
				sections = append(sections, cur)
				cur = section{heading: strings.TrimSpace(lines[i-1]), start: i - 1}
			}
		}
	}
	cur.end = len(lines)
	sections = append(sections, cur)

	// Join a section that is only a heading to the next one, so no passage is
	// a bare heading that ranks high and holds nothing.
	var merged []section
	for i := 0; i < len(sections); i++ {
		s := sections[i]
		for s.heading != "" && i+1 < len(sections) && !hasBody(lines, s) {
			s.end = sections[i+1].end
			if sections[i+1].heading != "" {
				s.heading = s.heading + " / " + sections[i+1].heading
			}
			i++
		}
		merged = append(merged, s)
	}

	var out []Passage
	for _, s := range merged {
		end := trimBlankEnd(lines, s.start, s.end)
		start := s.start
		for start < end && strings.TrimSpace(lines[start]) == "" {
			start++
		}
		if start >= end {
			continue
		}
		if end-start <= maxSectionLines {
			out = append(out, newPassage(lines, s.heading, start, end))
			continue
		}
		for w := start; w < end; w += windowLines {
			wEnd := min(w+windowLines, end)
			wEnd = trimBlankEnd(lines, w, wEnd)
			ws := w
			for ws < wEnd && strings.TrimSpace(lines[ws]) == "" {
				ws++
			}
			if ws < wEnd {
				out = append(out, newPassage(lines, s.heading, ws, wEnd))
			}
		}
	}
	return out
}

func newPassage(lines []string, heading string, start, end int) Passage {
	return Passage{
		Heading: heading,
		Start:   start + 1,
		End:     end,
		Text:    strings.Join(lines[start:end], "\n"),
	}
}

// frontmatterEnd returns the index of the first line after a leading YAML
// frontmatter block, or 0 when the file has none.
func frontmatterEnd(lines []string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1
		}
	}
	return 0
}

// hasBody reports whether a section has a non-blank line after its heading.
// For a setext heading the underline is part of the heading.
func hasBody(lines []string, s section) bool {
	from := s.start + 1
	if from < s.end && isSetextUnderline(lines[from]) {
		from++
	}
	for i := from; i < s.end; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return true
		}
	}
	return false
}

func trimBlankEnd(lines []string, start, end int) int {
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return end
}

// atxHeading returns the text of a `#` to `######` heading.
func atxHeading(line string) (string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return "", false
	}
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	if n == 0 || n > 6 {
		return "", false
	}
	rest := t[n:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", false
	}
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(rest), "#")), true
}

// isSetextUnderline reports whether a line is three or more `=` or three or
// more `-` and nothing else.
func isSetextUnderline(line string) bool {
	t := strings.TrimSpace(line)
	if len(t) < 3 {
		return false
	}
	c := t[0]
	if c != '=' && c != '-' {
		return false
	}
	return strings.Count(t, string(c)) == len(t)
}

// fenceMarker returns "```" or "~~~" when the line opens or closes a fenced
// code block.
func fenceMarker(line string) string {
	t := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(t, "```"):
		return "```"
	case strings.HasPrefix(t, "~~~"):
		return "~~~"
	}
	return ""
}
