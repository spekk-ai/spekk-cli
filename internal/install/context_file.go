package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A context-file host (gemini) has no per-skill directory: it auto-reads one
// shared markdown file — ~/.gemini/GEMINI.md globally, ./GEMINI.md in a project
// — and folds everything in it into the model's context. spekk cannot own the
// whole file, since the user keeps their own instructions there too, so it owns
// a single delimited section instead. These markers bound that section: install
// rewrites only what lies between them and preserves every other byte.
const (
	contextSectionBegin = "<!-- spekk:begin -->"
	contextSectionEnd   = "<!-- spekk:end -->"
)

// renderSpekkSection returns the full spekk-owned block, markers included and
// with no trailing newline, that install splices into a context file. It carries
// the observer, coach, and builder role instructions so a session seeded with
// the activation message ("Load and follow your spekk-coach skill") finds, in
// the auto-loaded context, a `### spekk-coach` heading whose body is the coach
// shim. The content is deterministic and FS-free, so Install and EnsureRoleSkill
// produce byte-identical sections and repeated installs are a no-op.
func renderSpekkSection() string {
	var b strings.Builder
	b.WriteString(contextSectionBegin)
	b.WriteString("\n\n## spekk\n\n")
	b.WriteString("This project uses spekk for spec-driven development. Each role below is an\n" +
		"agent you can act as; to adopt one, run its `spekk prompt <role>` command and\n" +
		"follow the printed instructions.\n")
	for _, role := range []string{"observer", "coach", "builder"} {
		b.WriteString("\n### spekk-")
		b.WriteString(role)
		b.WriteString("\n\n")
		b.WriteString(shimBody(role))
	}
	b.WriteByte('\n')
	b.WriteString(contextSectionEnd)
	return b.String()
}

// installContextFile writes (or refreshes) the spekk-owned section in the shared
// context file for the given scope. A scope the host cannot support resolves to
// "" and is a clean no-op rather than a wrong write.
func installContextFile(t target, project bool, home, cwd string) (Result, error) {
	path := t.contextFilePath(project, home, cwd)
	if path == "" {
		return Result{}, nil
	}
	return updateContextSection(path, renderSpekkSection())
}

// updateContextSection splices block into the context file at path, replacing an
// existing spekk section in place and preserving all other content. A write to a
// symlink or an unreadable path is refused with a warning, matching writeDesired.
// An already-current file is left untouched, so the call is idempotent.
func updateContextSection(path, block string) (Result, error) {
	var res Result
	found, err := inspect(path)
	if err != nil {
		return res, err
	}
	switch found.State {
	case pathSymlink:
		res.Warnings = append(res.Warnings, symlinkWarning(path, found.Link))
		return res, nil
	case pathOpaque:
		res.Warnings = append(res.Warnings, opaqueWarning(path))
		return res, nil
	}

	var existing []byte
	if found.State == pathReadable {
		existing = found.Content
	}
	updated := spliceSection(existing, block)
	if bytes.Equal(updated, existing) {
		return res, nil // already current: no-op (idempotent)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return res, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return res, fmt.Errorf("writing %s: %w", path, err)
	}
	res.Written = append(res.Written, path)
	return res, nil
}

// spliceSection returns existing with the spekk section set to block.
//
//   - No spekk section yet: block is appended, separated from any existing
//     content by a blank line, so it never runs into the user's own text.
//   - A spekk section is present: only the span from the begin marker through the
//     end marker is replaced; everything before and after is preserved verbatim.
//   - A begin marker with no end marker (a truncated section) is repaired: the
//     span from the begin marker to end-of-file is replaced.
func spliceSection(existing []byte, block string) []byte {
	b := bytes.Index(existing, []byte(contextSectionBegin))
	if b == -1 {
		if len(existing) == 0 {
			return []byte(block + "\n")
		}
		out := append([]byte{}, existing...)
		if out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, '\n')
		out = append(out, block...)
		out = append(out, '\n')
		return out
	}

	after := len(existing)
	if e := bytes.Index(existing[b:], []byte(contextSectionEnd)); e != -1 {
		after = b + e + len(contextSectionEnd)
	}
	out := append([]byte{}, existing[:b]...)
	out = append(out, block...)
	out = append(out, existing[after:]...)
	return out
}
