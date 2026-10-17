package docs

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// Stub is the frontmatter-only file in formatted/ that describes one source
// document. The summary exists so that an agent can decide from it alone
// whether to read the document.
type Stub struct {
	Title   string
	Type    string
	Created string
	Source  string
	Path    string
	Text    string
	SHA256  string
	Size    int64
	Tags    []string
	Summary string
}

// ReadStub reads a stub file. Hand edits are fine: a value may be quoted or
// bare, and tags may use any list spelling the spec parser accepts.
func ReadStub(p string) (Stub, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return Stub{}, err
	}
	f, err := parser.CustomFields(string(raw), nil)
	if err != nil {
		return Stub{}, err
	}
	s := Stub{
		Title:   scalar(f["title"]),
		Type:    scalar(f["type"]),
		Created: scalar(f["created"]),
		Source:  scalar(f["source"]),
		Path:    scalar(f["path"]),
		Text:    scalar(f["text"]),
		SHA256:  scalar(f["sha256"]),
		Tags:    f["tags"],
		Summary: scalar(f["summary"]),
	}
	if v := scalar(f["size"]); v != "" {
		s.Size, _ = strconv.ParseInt(v, 10, 64)
	}
	return s, nil
}

// Format renders the stub in a fixed key order. Strings are double-quoted
// with only `"` and `\` escaped, and newlines become spaces, so ReadStub
// gives back the same values.
func (s Stub) Format() string {
	var b strings.Builder
	b.WriteString("---\n")
	kv := func(k, v string) { fmt.Fprintf(&b, "%s: %s\n", k, quote(v)) }
	kv("title", s.Title)
	kv("type", s.Type)
	kv("created", s.Created)
	kv("source", s.Source)
	kv("path", s.Path)
	if s.Text != "" {
		kv("text", s.Text)
	}
	kv("sha256", s.SHA256)
	fmt.Fprintf(&b, "size: %d\n", s.Size)
	tags := make([]string, len(s.Tags))
	for i, t := range s.Tags {
		tags[i] = quote(t)
	}
	fmt.Fprintf(&b, "tags: [%s]\n", strings.Join(tags, ", "))
	kv("summary", s.Summary)
	b.WriteString("---\n")
	return b.String()
}

func quote(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return `"` + v + `"`
}
