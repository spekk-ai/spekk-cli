package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// marshalJSON encodes v as indented JSON without escaping HTML entities
// (matching Node's JSON.stringify behavior).
func marshalJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode appends a newline; trim it for consistency with MarshalIndent.
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// JSON output types matching the Node parser's output format.

// NextAssertionOutput is the JSON output for `spekk next` (single assertion).
type NextAssertionOutput struct {
	Type      string         `json:"type"`
	ID        string         `json:"id"`
	Parent    string         `json:"parent"`
	File      string         `json:"file"`
	Priority  int            `json:"priority"`
	Status    string         `json:"status"`
	Branch    string         `json:"branch"`
	Created   string         `json:"created"`
	DependsOn string         `json:"dependsOn,omitempty"`
	LockedBy  string         `json:"lockedBy,omitempty"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	Spec      *SpecRefOutput `json:"spec,omitempty"`
}

// SpecRefOutput is a reference to a parent spec within assertion output.
type SpecRefOutput struct {
	ID    string `json:"id"`
	File  string `json:"file"`
	Title string `json:"title"`
}

// HierarchyOutput is the JSON output for `spekk next --all`.
type HierarchyOutput struct {
	Type         string            `json:"type"`
	Specs        []HierarchySpec   `json:"specs"`
	Observations []json.RawMessage `json:"observations"`
}

// HierarchySpec is a spec entry within the hierarchy output.
type HierarchySpec struct {
	ID         string               `json:"id"`
	Title      string               `json:"title"`
	Status     string               `json:"status"`
	Priority   int                  `json:"priority"`
	File       string               `json:"file"`
	Assertions []HierarchyAssertion `json:"assertions"`
}

// HierarchyAssertion is an assertion entry within a hierarchy spec.
type HierarchyAssertion struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Priority  int      `json:"priority"`
	File      string   `json:"file"`
	DependsOn []string `json:"depends_on"`
}

// CompleteOutput is the JSON output when all specs are done.
type CompleteOutput struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// EmptyOutput is the JSON output when no specs are found.
type EmptyOutput struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorOutput is the JSON output for validation/parse errors.
type ErrorOutput struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// FormatNextAssertion formats a single assertion result as JSON.
func FormatNextAssertion(a *Assertion, specs []Spec) ([]byte, error) {
	var specRef *SpecRefOutput
	for _, s := range specs {
		if s.ID == a.Parent {
			specRef = &SpecRefOutput{
				ID:    s.ID,
				File:  s.File,
				Title: s.Title,
			}
			break
		}
	}

	out := NextAssertionOutput{
		Type:      "assertion",
		ID:        a.ID,
		Parent:    a.Parent,
		File:      a.File,
		Priority:  a.Priority,
		Status:    a.Status,
		Branch:    a.Branch,
		Created:   a.Created,
		DependsOn: a.DependsOn,
		LockedBy:  a.LockedBy,
		Title:     a.Title,
		Content:   a.Content,
		Spec:      specRef,
	}

	return marshalJSON(out)
}

// FormatHierarchy formats the full hierarchy as JSON.
func FormatHierarchy(result *ParseResult) ([]byte, error) {
	var hierarchySpecs []HierarchySpec

	for _, s := range result.Specs {
		var assertions []HierarchyAssertion
		for _, a := range result.Assertions {
			if a.Parent == s.ID {
				dependsOn := []string{}
				if a.DependsOn != "" {
					dependsOn = []string{a.DependsOn}
				}
				assertions = append(assertions, HierarchyAssertion{
					ID:        a.ID,
					Title:     a.Title,
					Status:    a.Status,
					Priority:  a.Priority,
					File:      a.File,
					DependsOn: dependsOn,
				})
			}
		}

		sort.Slice(assertions, func(i, j int) bool {
			if assertions[i].Priority != assertions[j].Priority {
				return assertions[i].Priority < assertions[j].Priority
			}
			return assertions[i].ID < assertions[j].ID
		})

		hierarchySpecs = append(hierarchySpecs, HierarchySpec{
			ID:         s.ID,
			Title:      s.Title,
			Status:     s.Status,
			Priority:   s.Priority,
			File:       s.File,
			Assertions: assertions,
		})
	}

	sort.Slice(hierarchySpecs, func(i, j int) bool {
		if hierarchySpecs[i].Priority != hierarchySpecs[j].Priority {
			return hierarchySpecs[i].Priority < hierarchySpecs[j].Priority
		}
		return hierarchySpecs[i].ID < hierarchySpecs[j].ID
	})

	out := HierarchyOutput{
		Type:         "hierarchy",
		Specs:        hierarchySpecs,
		Observations: []json.RawMessage{},
	}

	return marshalJSON(out)
}

// RawSpec is a spec in raw output format (all fields exposed).
type RawSpec struct {
	ID       string `json:"id"`
	Created  string `json:"created"`
	Priority int    `json:"priority"`
	Status   string `json:"status"`
	Branch   string `json:"branch"`
	File     string `json:"file"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

// RawAssertion is an assertion in raw output format (all fields exposed).
type RawAssertion struct {
	ID        string `json:"id"`
	Parent    string `json:"parent"`
	Created   string `json:"created"`
	Priority  int    `json:"priority"`
	Status    string `json:"status"`
	Branch    string `json:"branch"`
	DependsOn string `json:"dependsOn,omitempty"`
	LockedBy  string `json:"lockedBy,omitempty"`
	File      string `json:"file"`
	Title     string `json:"title"`
	Content   string `json:"content"`
}

// RawOutput is the raw parse result with all fields, used by downstream Node shims.
type RawOutput struct {
	Specs        []RawSpec         `json:"specs"`
	Assertions   []RawAssertion    `json:"assertions"`
	Observations []json.RawMessage `json:"observations"`
}

// FormatRaw formats the full parse result with all fields as JSON.
func FormatRaw(result *ParseResult) ([]byte, error) {
	rawSpecs := make([]RawSpec, len(result.Specs))
	for i, s := range result.Specs {
		rawSpecs[i] = RawSpec{
			ID: s.ID, Created: s.Created, Priority: s.Priority,
			Status: s.Status, Branch: s.Branch, File: s.File,
			Title: s.Title, Content: s.Content,
		}
	}

	rawAssertions := make([]RawAssertion, len(result.Assertions))
	for i, a := range result.Assertions {
		rawAssertions[i] = RawAssertion{
			ID: a.ID, Parent: a.Parent, Created: a.Created,
			Priority: a.Priority, Status: a.Status, Branch: a.Branch,
			DependsOn: a.DependsOn, LockedBy: a.LockedBy, File: a.File,
			Title: a.Title, Content: a.Content,
		}
	}

	out := RawOutput{
		Specs:        rawSpecs,
		Assertions:   rawAssertions,
		Observations: []json.RawMessage{},
	}

	return marshalJSON(out)
}

// FormatComplete formats the "all complete" output as JSON.
func FormatComplete() ([]byte, error) {
	out := CompleteOutput{
		Type:    "complete",
		Status:  "complete",
		Message: "All specifications are complete",
	}
	return marshalJSON(out)
}

// FormatEmpty formats the "no specs found" output as JSON.
func FormatEmpty() ([]byte, error) {
	out := EmptyOutput{
		Status:  "empty",
		Message: "No specifications found in specs/ directory",
	}
	return marshalJSON(out)
}

// FormatEmptyFiltered formats an empty-result message that mentions the active
// status filter, so callers can distinguish "no specs at all" from "filter
// matched nothing".
func FormatEmptyFiltered(status string) ([]byte, error) {
	out := EmptyOutput{
		Status:  "empty",
		Message: fmt.Sprintf("No assertions match status '%s'.", status),
	}
	return marshalJSON(out)
}

// FormatError formats an error as JSON.
func FormatError(msg string) ([]byte, error) {
	out := ErrorOutput{
		Error:   true,
		Message: msg,
	}
	return marshalJSON(out)
}

// FilterByStatus returns a new ParseResult containing only assertions with the
// given status, and only specs that have at least one matching assertion.
// Spec-level status is not used for filtering.
// Returns an error if status is not one of the values in validStatuses (parser.go).
func FilterByStatus(result *ParseResult, status string) (*ParseResult, error) {
	if !validStatuses[status] {
		keys := make([]string, 0, len(validStatuses))
		for k := range validStatuses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return nil, fmt.Errorf("invalid status %q — valid values: %s", status, strings.Join(keys, ", "))
	}

	var matchedAssertions []Assertion
	matchedParents := make(map[string]bool)
	for _, a := range result.Assertions {
		if a.Status == status {
			matchedAssertions = append(matchedAssertions, a)
			matchedParents[a.Parent] = true
		}
	}

	var matchedSpecs []Spec
	for _, s := range result.Specs {
		if matchedParents[s.ID] {
			matchedSpecs = append(matchedSpecs, s)
		}
	}

	return &ParseResult{
		Specs:      matchedSpecs,
		Assertions: matchedAssertions,
	}, nil
}

// AssertionsFlatOutput is the JSON output for `spekk list --assertions-only`.
type AssertionsFlatOutput struct {
	Type       string          `json:"type"`
	Assertions []FlatAssertion `json:"assertions"`
}

// FlatAssertion is an entry in the flat assertion list.
type FlatAssertion struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Priority  int      `json:"priority"`
	File      string   `json:"file"`
	Parent    string   `json:"parent"`
	DependsOn []string `json:"depends_on"`
}

// FormatAssertionsFlat produces a flat JSON list of all assertions in result,
// sorted by priority (ascending) then ID (alphabetical).
func FormatAssertionsFlat(result *ParseResult) ([]byte, error) {
	flat := make([]FlatAssertion, 0, len(result.Assertions))
	for _, a := range result.Assertions {
		dependsOn := []string{}
		if a.DependsOn != "" {
			dependsOn = []string{a.DependsOn}
		}
		flat = append(flat, FlatAssertion{
			ID:        a.ID,
			Title:     a.Title,
			Status:    a.Status,
			Priority:  a.Priority,
			File:      a.File,
			Parent:    a.Parent,
			DependsOn: dependsOn,
		})
	}

	sort.Slice(flat, func(i, j int) bool {
		if flat[i].Priority != flat[j].Priority {
			return flat[i].Priority < flat[j].Priority
		}
		return flat[i].ID < flat[j].ID
	})

	out := AssertionsFlatOutput{
		Type:       "assertions",
		Assertions: flat,
	}
	return marshalJSON(out)
}
