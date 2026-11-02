// Package anchors finds property anchors in source files selected by git.
package anchors

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Anchor names a property at a source line.
type Anchor struct {
	ID   string
	Path string
	Line int
}

var comment = regexp.MustCompile(`(?:#|//|--)[ \t]*spekk:[ \t]+([^ \t\r\n"'\\]+)[ \t]*$`)
var goTest = regexp.MustCompile(`^[ \t]*func[ \t]+(?:Test|Fuzz)[A-Za-z0-9_]*[ \t]*\(`)

// Scan reads tracked and untracked files that git does not ignore. The bool
// reports whether root is in a git work tree. Outside git it returns no anchors.
func Scan(root string) ([]Anchor, bool, error) {
	check := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	inside, err := check.Output()
	if err != nil || strings.TrimSpace(string(inside)) != "true" {
		return nil, false, nil
	}
	cmd := exec.Command("git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--")
	paths, err := cmd.Output()
	if err != nil {
		return nil, true, fmt.Errorf("cannot list source files: %w", err)
	}
	var found []Anchor
	for _, raw := range bytes.Split(paths, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		path := string(raw)
		if strings.HasSuffix(path, ".md") || hasTestdataSegment(path) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, path))
		if os.IsNotExist(err) { // A tracked file can be deleted in the work tree.
			continue
		}
		if err != nil {
			return nil, true, fmt.Errorf("cannot read %q: %w", path, err)
		}
		limit := len(content)
		if limit > 8000 {
			limit = 8000
		}
		if bytes.IndexByte(content[:limit], 0) >= 0 {
			continue
		}
		lines := bytes.Split(content, []byte{'\n'})
		for i, line := range lines {
			match := comment.FindSubmatch(line)
			if match == nil {
				continue
			}
			// A Go marker directly above a test function is evidence.
			if strings.HasSuffix(path, ".go") && i+1 < len(lines) && goTest.Match(lines[i+1]) {
				continue
			}
			found = append(found, Anchor{ID: string(match[1]), Path: filepath.ToSlash(path), Line: i + 1})
		}
	}
	return found, true, nil
}

func hasTestdataSegment(path string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == "testdata" {
			return true
		}
	}
	return false
}
