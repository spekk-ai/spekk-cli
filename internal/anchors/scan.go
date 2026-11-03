// Package anchors finds property anchors and evidence markers in source files selected by git.
package anchors

import (
	"bytes"
	"errors"
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

var comment = regexp.MustCompile(`(?:#|//|--)[ \t]*spekk:[ \t]+([^ \t\r\n"'\\]+)[ \t\r]*$`)
var goTest = regexp.MustCompile(`^[ \t]*func[ \t]+(Test[A-Z][A-Za-z0-9_]*)[ \t]*\([ \t]*[A-Za-z_][A-Za-z0-9_]*[ \t]*\*[ \t]*testing\.T[ \t]*\)`)
var goFuzz = regexp.MustCompile(`^[ \t]*func[ \t]+(Fuzz[A-Z][A-Za-z0-9_]*)[ \t]*\([ \t]*[A-Za-z_][A-Za-z0-9_]*[ \t]*\*[ \t]*testing\.F[ \t]*\)`)
var goMarker = regexp.MustCompile(`^[ \t]*//[ \t]*spekk:[ \t]+([^ \t\r\n]+)(?:[ \t]+evidence=([^ \t\r\n]+))?[ \t\r]*$`)
var pythonMarker = regexp.MustCompile(`^[ \t]*@spekk\([ \t]*(['"])([^'"]+)['"](.*)\)[ \t\r]*$`)
var pythonEvidence = regexp.MustCompile(`(?:^|,)[ \t]*evidence[ \t]*=[ \t]*['"]([^'"]+)['"]`)
var pythonDef = regexp.MustCompile(`^[ \t]*def[ \t]+([A-Za-z_][A-Za-z0-9_]*)[ \t]*\(`)
var pythonClass = regexp.MustCompile(`^[ \t]*class[ \t]+(Test[A-Za-z0-9_]*)\b`)

// Evidence links one test to one property.
type Evidence struct {
	ID, Level, Language, Path, Test string
	Line                            int
}

// Warning names a decorator that cannot mark a pytest test.
type Warning struct {
	Path string
	Line int
}

// Result holds every source annotation from one file walk.
type Result struct {
	Anchors  []Anchor
	Evidence []Evidence
	Warnings []Warning
}

// ScanAll reads tracked and untracked files that git does not ignore. The bool reports whether root is in a git work tree.
func ScanAll(root string) (Result, bool, error) {
	check := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	inside, err := check.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || bytes.Contains(inside, []byte("not a git repository")) {
			return Result{}, false, nil
		}
		return Result{}, false, fmt.Errorf("cannot check git work tree: %w: %s", err, bytes.TrimSpace(inside))
	}
	if strings.TrimSpace(string(inside)) != "true" {
		return Result{}, false, nil
	}
	cmd := exec.Command("git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--")
	paths, err := cmd.Output()
	if err != nil {
		return Result{}, true, fmt.Errorf("cannot list source files: %w", err)
	}
	var found Result
	for _, raw := range bytes.Split(paths, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		path := string(raw)
		if strings.HasSuffix(path, ".md") || hasTestdataSegment(path) {
			continue
		}
		fullPath := filepath.Join(root, path)
		info, err := os.Lstat(fullPath)
		if os.IsNotExist(err) { // A tracked file can be deleted in the work tree.
			continue
		}
		if err != nil {
			return Result{}, true, fmt.Errorf("cannot stat %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		content, err := os.ReadFile(fullPath)
		if os.IsNotExist(err) { // A tracked file can be deleted in the work tree.
			continue
		}
		if err != nil {
			return Result{}, true, fmt.Errorf("cannot read %q: %w", path, err)
		}
		limit := len(content)
		if limit > 8000 {
			limit = 8000
		}
		if bytes.IndexByte(content[:limit], 0) >= 0 {
			continue
		}
		lines := strings.Split(string(content), "\n")
		if strings.HasSuffix(path, "_test.go") {
			scanGoMarkers(&found, path, lines)
		}
		if isPythonTestFile(path) {
			scanPythonMarkers(&found, path, lines)
		}
		for i, line := range lines {
			match := comment.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			if strings.HasSuffix(path, "_test.go") && goMarker.MatchString(line) && goMarkerFunction(lines, i) != "" {
				continue
			}
			found.Anchors = append(found.Anchors, Anchor{ID: match[1], Path: filepath.ToSlash(path), Line: i + 1})
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

// Head returns the current commit, or an empty value before the first commit or outside a git work tree.
func Head(root string) (string, error) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--verify", "HEAD").CombinedOutput()
	if err != nil {
		if bytes.Contains(out, []byte("not a git repository")) || bytes.Contains(out, []byte("Needed a single revision")) {
			return "", nil
		}
		return "", fmt.Errorf("cannot read git HEAD: %w: %s", err, bytes.TrimSpace(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func goMarkerFunction(lines []string, marker int) string {
	for i := marker + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "//") {
			continue
		}
		if match := goTest.FindStringSubmatch(lines[i]); match != nil {
			return match[1]
		}
		if match := goFuzz.FindStringSubmatch(lines[i]); match != nil {
			return match[1]
		}
		return ""
	}
	return ""
}

func scanGoMarkers(found *Result, path string, lines []string) {
	for i, line := range lines {
		match := goMarker.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		test := goMarkerFunction(lines, i)
		if test == "" {
			continue
		}
		level := match[2]
		if level == "" {
			level = "example"
		}
		found.Evidence = append(found.Evidence, Evidence{ID: match[1], Level: level, Language: "go", Path: filepath.ToSlash(path), Line: i + 1, Test: test})
	}
}

func isPythonTestFile(path string) bool {
	name := filepath.Base(path)
	return (strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test.py")) && strings.HasSuffix(name, ".py")
}

func indentation(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func scanPythonMarkers(found *Result, path string, lines []string) {
	type decorator struct {
		id, level string
		line      int
	}
	var pending []decorator
	className := ""
	classIndent := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := indentation(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			pending = nil
			continue
		}
		if className != "" && indent <= classIndent {
			className = ""
			classIndent = -1
		}
		if match := pythonClass.FindStringSubmatch(line); match != nil {
			className, classIndent = match[1], indent
			pending = nil
			continue
		}
		if match := pythonMarker.FindStringSubmatch(line); match != nil {
			level := "example"
			if value := pythonEvidence.FindStringSubmatch(match[3]); value != nil {
				level = value[1]
			}
			pending = append(pending, decorator{id: match[2], level: level, line: i + 1})
			continue
		}
		if strings.HasPrefix(trimmed, "@") {
			continue
		}
		if match := pythonDef.FindStringSubmatch(line); match != nil {
			name := match[1]
			valid := strings.HasPrefix(name, "test") && (indent == 0 || (className != "" && indent > classIndent))
			for _, marker := range pending {
				if !valid {
					found.Warnings = append(found.Warnings, Warning{Path: filepath.ToSlash(path), Line: marker.line})
					continue
				}
				test := name
				if className != "" {
					test = className + "." + name
				}
				found.Evidence = append(found.Evidence, Evidence{ID: marker.id, Level: marker.level, Language: "python", Path: filepath.ToSlash(path), Line: marker.line, Test: test})
			}
		}
		pending = nil
	}
}
