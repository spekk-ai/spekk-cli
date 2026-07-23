// Package validate implements the `spekk validate` command: a strict,
// non-zero-exit gate over the invariants that internal/parser's working-tree
// parser only enforces leniently (skip-with-warning) or not at all (lock-state
// pairing, parent-status legality).
//
// It reuses internal/parser's exported per-file parsing (ParseSpecContent /
// ParseAssertionContent) for frontmatter well-formedness rather than
// reimplementing a frontmatter parser. What validate adds on top:
//   - promotes the parser's "skip with a warning" outcomes (malformed
//     frontmatter, duplicate assertion ids) to hard failures,
//   - checks lock-state pairing (in_progress <=> locked-by),
//   - checks that parent spec files carry no rolled-up status field (or only
//     the literal value "draft"),
//   - and reports every violation found in one pass (not just the first),
//     sorted deterministically for stable, diffable output.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// Failure is a single validation violation, tied to the file it was found in.
type Failure struct {
	File    string
	Message string
}

// String renders a failure as a single plain-text line:
// "<file>: <message>".
func (f Failure) String() string {
	return fmt.Sprintf("%s: %s", f.File, f.Message)
}

// Result is the outcome of validating a specs/ tree.
type Result struct {
	Failures       []Failure
	SpecCount      int
	AssertionCount int
}

// Passed reports whether validation found zero failures.
func (r *Result) Passed() bool {
	return len(r.Failures) == 0
}

// kebabCasePattern matches valid kebab-case identifiers. Kept in sync with
// internal/parser's (unexported) pattern of the same name; depends-on is not
// validated by the parser's per-file parse functions, so validate checks it
// directly as a relational invariant, not a frontmatter field.
var kebabCasePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Run walks specsDir and validates every invariant in specs/spec-validation.
// It never returns an error for a merely-invalid tree — problems are reported
// as Failures. A returned error indicates specsDir itself could not be read.
func Run(specsDir string) (*Result, error) {
	result := &Result{}

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("cannot read specs directory %q: %w", specsDir, err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var specs []parser.Spec
	var assertions []parser.Assertion

	specFilesByID := map[string][]string{}
	assertionFilesByID := map[string][]string{}

	for _, entry := range entries {
		if !entry.IsDir() {
			// Flat .md files directly under specs/ are out of scope for the
			// per-spec-directory invariants below; the parser already hard-fails
			// this case structurally for spekk next/list.
			continue
		}

		specDirName := entry.Name()
		specDirPath := filepath.Join(specsDir, specDirName)
		specFilePath := filepath.Join(specDirPath, specDirName+".md")
		assertionsDirPath := filepath.Join(specDirPath, "assertions")

		relSpecFilePath := filepath.ToSlash(filepath.Join("specs", specDirName, specDirName+".md"))

		if raw, readErr := os.ReadFile(specFilePath); readErr == nil {
			content := string(raw)
			if hasFrontmatter(content) {
				relPath := relSpecFilePath
				spec, parseErr := parser.ParseSpecContent(relPath, content)
				if parseErr != nil {
					result.addFailure(relPath, "frontmatter: "+parseErr.Error())
				} else {
					specFilesByID[spec.ID] = append(specFilesByID[spec.ID], relPath)
					specs = append(specs, *spec)

					if present, value := rawStatusField(content); present && value != "draft" {
						result.addFailure(relPath, fmt.Sprintf(
							"parent spec has disallowed status field %q (must be absent, or the literal value \"draft\")", value))
					}
				}
			}
		}

		assertionEntries, readErr := os.ReadDir(assertionsDirPath)
		if readErr != nil {
			continue
		}
		sort.Slice(assertionEntries, func(i, j int) bool {
			return assertionEntries[i].Name() < assertionEntries[j].Name()
		})

		for _, aEntry := range assertionEntries {
			if aEntry.IsDir() || !strings.HasSuffix(aEntry.Name(), ".md") {
				continue
			}
			aFilePath := filepath.Join(assertionsDirPath, aEntry.Name())
			relAPath := filepath.ToSlash(filepath.Join("specs", specDirName, "assertions", aEntry.Name()))

			raw, readErr := os.ReadFile(aFilePath)
			if readErr != nil {
				result.addFailure(relAPath, "cannot read file: "+readErr.Error())
				continue
			}
			content := string(raw)
			if !hasFrontmatter(content) {
				// Not every .md file is a spec file; files with no frontmatter are ignored.
				continue
			}

			assertion, parseErr := parser.ParseAssertionContent(relAPath, content)
			if parseErr != nil {
				result.addFailure(relAPath, "frontmatter: "+parseErr.Error())
				continue
			}

			assertionFilesByID[assertion.ID] = append(assertionFilesByID[assertion.ID], relAPath)
			assertions = append(assertions, *assertion)

			checkLockState(*assertion, result)
		}
	}

	checkDuplicateIDs(specFilesByID, "spec", result)
	checkDuplicateIDs(assertionFilesByID, "assertion", result)

	specIDs := make(map[string]bool, len(specFilesByID))
	for id := range specFilesByID {
		specIDs[id] = true
	}
	assertionIDs := make(map[string]bool, len(assertionFilesByID))
	for id := range assertionFilesByID {
		assertionIDs[id] = true
	}

	for _, a := range assertions {
		if !specIDs[a.Parent] {
			result.addFailure(a.File, fmt.Sprintf("parent %q not found", a.Parent))
		}
	}

	checkDependsOn(assertions, assertionIDs, result)
	checkCycles(assertions, result)

	result.SpecCount = len(specs)
	result.AssertionCount = len(assertions)

	sort.Slice(result.Failures, func(i, j int) bool {
		if result.Failures[i].File != result.Failures[j].File {
			return result.Failures[i].File < result.Failures[j].File
		}
		return result.Failures[i].Message < result.Failures[j].Message
	})

	return result, nil
}

func (r *Result) addFailure(file, message string) {
	r.Failures = append(r.Failures, Failure{File: file, Message: message})
}

// hasFrontmatter reports whether file content starts with a frontmatter
// delimiter (after CRLF normalisation). Files without frontmatter are not
// spec files and are silently ignored, matching the lenient parser.
func hasFrontmatter(content string) bool {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.HasPrefix(normalized, "---")
}

// rawStatusField reports whether the literal "status:" key is present in a
// file's YAML frontmatter block, and its raw value if so. This is a
// key-presence/value check over the raw text — not a second frontmatter
// parser — needed because ParseSpecContent defaults an *absent* status to
// "not_started", making the parsed model unable to distinguish "no status
// field" (legal on a parent) from "status: not_started" (illegal).
func rawStatusField(content string) (present bool, value string) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return false, ""
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "status:") {
			continue
		}
		value = strings.TrimSpace(strings.TrimPrefix(trimmed, "status:"))
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}
		return true, value
	}
	return false, ""
}

// checkLockState enforces: status in_progress requires a non-empty
// locked-by; every other status (done, failed, not_started, draft) forbids
// one.
func checkLockState(a parser.Assertion, result *Result) {
	switch a.Status {
	case "in_progress":
		if strings.TrimSpace(a.LockedBy) == "" {
			result.addFailure(a.File, "status is in_progress but locked-by is missing")
		}
	default:
		if a.LockedBy != "" {
			result.addFailure(a.File, fmt.Sprintf("status is %s but locked-by is set (%q); only in_progress may carry a lock", a.Status, a.LockedBy))
		}
	}
}

// checkDuplicateIDs reports every file beyond the first that declares a
// given id, for either specs or assertions. The lenient parser only warns
// (and skips) on duplicate assertion ids and hard-fails on the first
// duplicate spec id found; validate reports every offending file instead.
func checkDuplicateIDs(filesByID map[string][]string, kind string, result *Result) {
	for id, files := range filesByID {
		if len(files) < 2 {
			continue
		}
		sorted := append([]string(nil), files...)
		sort.Strings(sorted)
		for _, f := range sorted {
			others := make([]string, 0, len(sorted)-1)
			for _, other := range sorted {
				if other != f {
					others = append(others, other)
				}
			}
			result.addFailure(f, fmt.Sprintf("duplicate %s id %q also declared in %s", kind, id, strings.Join(others, ", ")))
		}
	}
}

// checkDependsOn validates every non-empty depends-on field: kebab-case,
// references an existing assertion, and is not self-referential. Cycle
// detection is handled separately by checkCycles.
func checkDependsOn(assertions []parser.Assertion, assertionIDs map[string]bool, result *Result) {
	for _, a := range assertions {
		if a.DependsOn == "" {
			continue
		}
		if !kebabCasePattern.MatchString(a.DependsOn) {
			result.addFailure(a.File, fmt.Sprintf("depends-on %q is not kebab-case", a.DependsOn))
			continue
		}
		if a.DependsOn == a.ID {
			result.addFailure(a.File, "depends-on cannot reference itself")
			continue
		}
		if !assertionIDs[a.DependsOn] {
			result.addFailure(a.File, fmt.Sprintf("depends-on references non-existent assertion %q", a.DependsOn))
		}
	}
}

// checkCycles reports every assertion that participates in a depends-on
// cycle, one failure per participant, so a cyclic mistake is visible no
// matter which file in the cycle a reader opens.
func checkCycles(assertions []parser.Assertion, result *Result) {
	dependsOn := make(map[string]string, len(assertions))
	fileByID := make(map[string]string, len(assertions))
	for _, a := range assertions {
		if a.DependsOn != "" {
			dependsOn[a.ID] = a.DependsOn
		}
		fileByID[a.ID] = a.File
	}

	reported := map[string]bool{}
	for _, a := range assertions {
		if a.DependsOn == "" || reported[a.ID] {
			continue
		}
		// Self-references are a degenerate cycle already reported by
		// checkDependsOn with a clearer message; don't double-report them here.
		if a.DependsOn == a.ID {
			continue
		}

		visited := map[string]bool{}
		var path []string
		current := a.ID
		for current != "" {
			if visited[current] {
				cycleStart := -1
				for i, id := range path {
					if id == current {
						cycleStart = i
						break
					}
				}
				cycle := path[cycleStart:]
				cyclePath := strings.Join(append(append([]string{}, cycle...), current), " -> ")
				for _, id := range cycle {
					if reported[id] {
						continue
					}
					reported[id] = true
					if f, ok := fileByID[id]; ok {
						result.addFailure(f, fmt.Sprintf("depends-on cycle: %s", cyclePath))
					}
				}
				break
			}
			visited[current] = true
			path = append(path, current)
			current = dependsOn[current]
		}
	}
}
