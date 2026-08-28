// Package parser provides spec assertion parsing logic for the spekk CLI.
package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Assertion represents a parsed spec assertion with its frontmatter and content.
type Assertion struct {
	ID        string
	Parent    string
	Created   string
	Priority  int
	Status    string
	Branch    string
	DependsOn string
	LockedBy  string
	File      string
	Title     string
	Content   string
	// Fields holds custom frontmatter keys (everything outside the known
	// set) with their parsed values, one entry per item — every list
	// spelling (flow sequence, comma scalar, block list) arrives in the
	// same shape. nil when the file has none.
	Fields map[string][]string
}

// Spec represents a parsed spec group with its frontmatter and content.
type Spec struct {
	ID       string
	Created  string
	Priority int
	Status   string
	Branch   string
	File     string
	Title    string
	Content  string
	// Fields holds custom frontmatter keys (everything outside the known
	// set) with their parsed values, one entry per item — every list
	// spelling (flow sequence, comma scalar, block list) arrives in the
	// same shape. nil when the file has none.
	Fields map[string][]string
}

// ParseResult holds the full result of parsing a specs directory.
type ParseResult struct {
	Specs      []Spec
	Assertions []Assertion

	// Warnings records every spec or assertion file the walk skipped, one
	// entry per file, in the order the walk met them. The parser does not
	// print them: a caller that shows output decides whether a user sees a
	// summary, and a caller that only rebuilds the index says nothing. While
	// the parse printed them itself, spekk next emitted every warning twice,
	// because it parses once for the index and once to answer.
	Warnings []string
}

// frontmatter holds raw parsed YAML frontmatter fields. Scalar values live
// in fields (double quotes stripped), so known-key handling stays
// byte-for-byte unchanged. The custom-field layer reads from raw and lists
// instead: raw keeps the unstripped scalar of every top-level key (so a
// quoted scalar is still recognizable as one value), and lists holds
// block-list items (`- foo` lines under a bare `key:`).
type frontmatter struct {
	fields map[string]string
	raw    map[string]string
	lists  map[string][]string
}

// knownFrontmatterKeys are the keys the parser maps onto Spec/Assertion
// struct fields. Every other key is a custom field, preserved on Fields.
var knownFrontmatterKeys = map[string]bool{
	"id":         true,
	"parent":     true,
	"created":    true,
	"priority":   true,
	"status":     true,
	"branch":     true,
	"depends-on": true,
	"locked-by":  true,
}

func (f *frontmatter) get(key string) string {
	return f.fields[key]
}

func (f *frontmatter) getInt(key string) (int, error) {
	v, ok := f.fields[key]
	if !ok || v == "" {
		return 0, fmt.Errorf("missing field %q", key)
	}
	return strconv.Atoi(v)
}

// parseFrontmatter parses a markdown file's YAML frontmatter block.
// Returns the frontmatter, the markdown body after closing ---, and any error.
// Files not starting with --- return an error; callers can choose to skip them.
func parseFrontmatter(content string) (*frontmatter, string, error) {
	// Normalize CRLF to LF.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	lines := strings.Split(content, "\n")

	if len(lines) == 0 || lines[0] != "---" {
		return nil, "", fmt.Errorf("file must start with --- YAML frontmatter delimiter")
	}

	frontmatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatterEnd = i
			break
		}
	}

	if frontmatterEnd == -1 {
		return nil, "", fmt.Errorf("missing closing --- delimiter for YAML frontmatter")
	}

	yamlLines := lines[1:frontmatterEnd]
	markdownContent := strings.Join(lines[frontmatterEnd+1:], "\n")

	fm := &frontmatter{
		fields: make(map[string]string),
		raw:    make(map[string]string),
		lists:  make(map[string][]string),
	}

	// The key a following block-list item belongs to: the most recent
	// top-level `key:` line with an empty value.
	pendingListKey := ""

	for _, line := range yamlLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// A YAML block-list item. Collect it under the key that opened the
		// list. The item is kept whole — commas inside an item never split.
		if strings.HasPrefix(trimmed, "- ") {
			if pendingListKey != "" {
				item := stripQuotes(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
				if item != "" {
					fm.lists[pendingListKey] = append(fm.lists[pendingListKey], item)
				}
			}
			continue
		}
		// A YAML comment. Invisible, and it does not interrupt a block list.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// An indented line is a nested child (nested map, block-scalar
		// body). It must not become a top-level custom field or close an
		// open block list. The fields map still records it for byte-for-byte
		// compatibility with the known-key scanner.
		indented := line[0] == ' ' || line[0] == '\t'

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			if !indented {
				pendingListKey = ""
			}
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		if !indented {
			if value == "" {
				pendingListKey = key
			} else {
				pendingListKey = ""
			}
			if key != "" {
				fm.raw[key] = value
			}
		}

		// Strip surrounding quotes if present.
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		fm.fields[key] = value
	}

	return fm, markdownContent, nil
}

// CustomFields parses content's YAML frontmatter and returns every
// top-level key outside known, with its values already split into items.
// It is the shared entry point for a file type whose known keys are not a
// spec's — an observation, say — so that every owner type indexes custom
// fields under one rule instead of one copy of it per format.
func CustomFields(content string, known map[string]bool) (map[string][]string, error) {
	fm, _, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}
	return customFields(fm, known), nil
}

// stripQuotes removes one matching pair of surrounding double or single
// quotes.
func stripQuotes(s string) string {
	if len(s) >= 2 &&
		((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// customFields returns every top-level frontmatter key outside the known
// set, with its values already split into items. Three list spellings
// produce identical items: a flow sequence (`[a, b]`), a bare
// comma-separated scalar (`a, b`), and a block list (whose items are never
// re-split, so an item may contain commas). Nested-map children, comments,
// empty keys, and block-scalar bodies never appear. Returns nil when the
// file has no custom fields.
func customFields(fm *frontmatter, known map[string]bool) map[string][]string {
	out := make(map[string][]string)
	for k, raw := range fm.raw {
		if known[k] {
			continue
		}
		var values []string
		if items, ok := fm.lists[k]; ok && raw == "" {
			values = items
		} else {
			values = splitFieldValues(raw)
		}
		if len(values) > 0 {
			out[k] = values
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// blockScalarIndicators are scalar values that announce a YAML block scalar
// (`key: |`). The line scanner cannot read the indented body, so the value
// carries no data — customFields drops the key instead of indexing "|".
var blockScalarIndicators = map[string]bool{
	"|": true, "|-": true, "|+": true,
	">": true, ">-": true, ">+": true,
}

// splitFieldValues splits a custom frontmatter scalar into its items. A
// fully quoted scalar is one item. A flow sequence loses its brackets and
// splits on commas outside quotes; a bare scalar splits on commas outside
// quotes. Items are trimmed and one pair of surrounding quotes is removed
// per item. An empty value or a block-scalar indicator yields nil.
func splitFieldValues(value string) []string {
	v := strings.TrimSpace(value)
	if v == "" || blockScalarIndicators[v] {
		return nil
	}
	if isOneQuotedScalar(v) {
		return []string{v[1 : len(v)-1]}
	}
	if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") && len(v) >= 2 {
		v = v[1 : len(v)-1]
	}
	var out []string
	for _, part := range splitOutsideQuotes(v, ',') {
		p := stripQuotes(strings.TrimSpace(part))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isOneQuotedScalar reports whether s is a single quoted string — quoted at
// both ends with no inner occurrence of the same quote, so `"Hello, world"`
// is one value while `"a", "b"` is not.
func isOneQuotedScalar(s string) bool {
	if len(s) < 2 {
		return false
	}
	q := s[0]
	if (q != '"' && q != '\'') || s[len(s)-1] != q {
		return false
	}
	return !strings.ContainsRune(s[1:len(s)-1], rune(q))
}

// splitOutsideQuotes splits s on sep, ignoring separators inside single- or
// double-quoted regions, so a quoted item keeps its commas.
func splitOutsideQuotes(s string, sep byte) []string {
	var parts []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			cur.WriteByte(c)
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
			cur.WriteByte(c)
		case c == sep:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	parts = append(parts, cur.String())
	return parts
}

// extractTitle finds the first H1 heading in markdown content.
// Returns "Untitled" if no H1 is found.
func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}

// timestampPattern matches ISO 8601 timestamps of the form YYYY-MM-DDTHH:MM:SSZ.
var timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)

func validateTimestamp(value string) bool {
	return timestampPattern.MatchString(value)
}

// kebabCasePattern matches valid kebab-case identifiers.
var kebabCasePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// IsKebabCase reports whether s is a valid spec or assertion identifier. It is
// the one definition of that rule, so a caller outside this package checks the
// same thing the parser does instead of copying the pattern.
func IsKebabCase(s string) bool {
	return kebabCasePattern.MatchString(s)
}

// validBranchPattern matches valid git branch name characters.
//
// A dot is permitted, because git permits it and a release branch usually
// carries a version. Git's own rules for dots are applied in validateBranch,
// because a character class cannot express them.
var validBranchPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9/._-]*$`)

// validateBranch reports a branch value that git itself would refuse. It
// judges no naming convention.
//
// A list of accepted <type>/ prefixes stood here, and warned on every other
// shape. It was a proxy for "this value names no branch", and a poor one: it
// passed a typo such as feat/thing-nmae, which names nothing, and it warned on
// dana/apx-12-thing, which git accepts and a team uses. The real check reads
// the refs, so it belongs to validate, not to a pure per-file parse that runs
// once per file. See internal/validate.
func validateBranch(branch, filePath string) error {
	if branch == "" {
		return nil
	}
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("Field 'branch' cannot start or end with '/' in %s", filePath)
	}
	if !validBranchPattern.MatchString(branch) {
		return fmt.Errorf("Field 'branch' contains invalid characters in %s\nFound: %q\nGit branch names can only contain letters, numbers, slashes, dots, hyphens, and underscores.", filePath, branch)
	}
	// Git's rules for dots, which a character class cannot express: no "..",
	// no final ".", and no ".lock" suffix. These checks keep the field to
	// names that git accepts.
	if strings.Contains(branch, "..") || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, ".lock") {
		return fmt.Errorf("Field 'branch' is not a valid git branch name in %s\nFound: %q\nA branch name cannot contain '..', end with '.', or end with '.lock'.", filePath, branch)
	}
	return nil
}

var validStatuses = map[string]bool{
	"not_started": true,
	"in_progress": true,
	"done":        true,
	"draft":       true,
	"failed":      true,
}

// parseSpec parses a spec parent file (e.g., specs/foo/foo.md).
func parseSpec(relFilePath string, content string) (*Spec, error) {
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	id := fm.get("id")
	if id == "" {
		return nil, fmt.Errorf("missing required field 'id'")
	}
	if !kebabCasePattern.MatchString(id) {
		return nil, fmt.Errorf("invalid id format %q (must be kebab-case: lowercase with hyphens, no spaces/underscores/special chars)", id)
	}

	created := fm.get("created")
	if created == "" {
		return nil, fmt.Errorf("missing required field 'created'")
	}
	if !validateTimestamp(created) {
		return nil, fmt.Errorf("invalid ISO 8601 timestamp in 'created' field: %q", created)
	}

	priority, err := fm.getInt("priority")
	if err != nil {
		return nil, fmt.Errorf("missing or invalid required field 'priority'")
	}
	if priority < 1 || priority > 3 {
		return nil, fmt.Errorf("invalid priority value %d (must be 1, 2, or 3)", priority)
	}

	status := fm.get("status")
	if status == "" {
		status = "not_started"
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status value %q", status)
	}

	branch := fm.get("branch")
	if branch == "" {
		branch = "main"
	}
	if err := validateBranch(branch, relFilePath); err != nil {
		return nil, err
	}

	return &Spec{
		ID:       id,
		Created:  created,
		Priority: priority,
		Status:   status,
		Branch:   branch,
		File:     relFilePath,
		Title:    extractTitle(body),
		Content:  strings.TrimSpace(body),
		Fields:   customFields(fm, knownFrontmatterKeys),
	}, nil
}

// parseAssertion parses an assertion file (e.g., specs/foo/assertions/bar.md).
func parseAssertion(relFilePath string, content string) (*Assertion, error) {
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	id := fm.get("id")
	if id == "" {
		return nil, fmt.Errorf("missing required field 'id'")
	}
	if !kebabCasePattern.MatchString(id) {
		return nil, fmt.Errorf("invalid id format %q (must be kebab-case: lowercase with hyphens, no spaces/underscores/special chars)", id)
	}

	parent := fm.get("parent")
	if parent == "" {
		return nil, fmt.Errorf("missing required field 'parent'")
	}

	created := fm.get("created")
	if created == "" {
		return nil, fmt.Errorf("missing required field 'created'")
	}
	if !validateTimestamp(created) {
		return nil, fmt.Errorf("invalid ISO 8601 timestamp in 'created' field: %q", created)
	}

	priority, err := fm.getInt("priority")
	if err != nil {
		return nil, fmt.Errorf("missing or invalid required field 'priority'")
	}
	if priority < 1 || priority > 3 {
		return nil, fmt.Errorf("invalid priority value %d (must be 1, 2, or 3)", priority)
	}

	status := fm.get("status")
	if status == "" {
		status = "not_started"
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status value %q", status)
	}

	branch := fm.get("branch")
	if branch == "" {
		branch = "main"
	}
	if err := validateBranch(branch, relFilePath); err != nil {
		return nil, err
	}

	return &Assertion{
		ID:        id,
		Parent:    parent,
		Created:   created,
		Priority:  priority,
		Status:    status,
		Branch:    branch,
		DependsOn: fm.get("depends-on"),
		LockedBy:  fm.get("locked-by"),
		File:      relFilePath,
		Title:     extractTitle(body),
		Content:   strings.TrimSpace(body),
		Fields:    customFields(fm, knownFrontmatterKeys),
	}, nil
}

// ParseSpecContent parses spec parent file content (e.g. the text of
// specs/foo/foo.md) that has been read from somewhere other than disk — such as
// the bytes of a file at a particular git ref. relFilePath is used only for
// diagnostics; it does not need to exist on the filesystem.
//
// It is a thin exported wrapper over the same per-file parse logic used for
// working-tree files, so callers (e.g. the cross-branch explorer) reuse the
// existing frontmatter/title/status parsing rather than re-implementing it.
func ParseSpecContent(relFilePath string, content string) (*Spec, error) {
	return parseSpec(relFilePath, content)
}

// ParseAssertionContent parses assertion file content (e.g. the text of
// specs/foo/assertions/bar.md) that has been read from somewhere other than
// disk — such as the bytes of a file at a particular git ref. relFilePath is
// used only for diagnostics; it does not need to exist on the filesystem.
//
// It is a thin exported wrapper over the same per-file parse logic used for
// working-tree files, so callers (e.g. the cross-branch explorer) reuse the
// existing frontmatter/title/status parsing rather than re-implementing it.
func ParseAssertionContent(relFilePath string, content string) (*Assertion, error) {
	return parseAssertion(relFilePath, content)
}

// hasFrontmatter reports whether file content starts with a frontmatter delimiter
// (after CRLF normalisation).
func hasFrontmatter(content string) bool {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.HasPrefix(normalized, "---")
}

// ParseAllSpecs walks specsDir and parses all specs and assertions.
// Malformed or incomplete structures produce warnings on stderr and are skipped.
func ParseAllSpecs(specsDir string) (*ParseResult, error) {
	info, err := os.Stat(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &ParseResult{}, nil
		}
		return nil, fmt.Errorf("cannot access specs directory %q: %w", specsDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", specsDir)
	}

	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read specs directory %q: %w", specsDir, err)
	}

	// Sort entries for deterministic ordering.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var specs []Spec
	var assertions []Assertion
	var warnings []string

	specIDsSeen := make(map[string]string) // id -> relative file path

	for _, entry := range entries {
		if !entry.IsDir() {
			// Flat .md files at specs/ level with frontmatter are an error.
			if strings.HasSuffix(entry.Name(), ".md") {
				flatPath := filepath.Join(specsDir, entry.Name())
				raw, readErr := os.ReadFile(flatPath)
				if readErr == nil && hasFrontmatter(string(raw)) {
					return nil, fmt.Errorf("Invalid folder structure: Found flat .md files with frontmatter in specs/: %s. All specs must be in folders following the pattern specs/{spec-id}/{spec-id}.md", entry.Name())
				}
			}
			continue
		}

		specDirName := entry.Name()
		specDirPath := filepath.Join(specsDir, specDirName)
		specFilePath := filepath.Join(specDirPath, specDirName+".md")
		assertionsDirPath := filepath.Join(specDirPath, "assertions")

		// Determine whether this directory contains any actual spec files.
		hasSpecFiles := false

		specFileRaw, readErr := os.ReadFile(specFilePath)
		if readErr == nil && hasFrontmatter(string(specFileRaw)) {
			hasSpecFiles = true
		}

		if !hasSpecFiles {
			// Check assertions directory for frontmatter files.
			_ = filepath.WalkDir(assertionsDirPath, func(p string, d fs.DirEntry, werr error) error {
				if werr != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
					return nil
				}
				raw, rerr := os.ReadFile(p)
				if rerr == nil && hasFrontmatter(string(raw)) {
					hasSpecFiles = true
				}
				return nil
			})
		}

		if !hasSpecFiles {
			// Not an actual spec group directory — skip silently.
			continue
		}

		// The main spec file must exist.
		if _, statErr := os.Stat(specFilePath); os.IsNotExist(statErr) {
			warnings = append(warnings, fmt.Sprintf("Spec %s/ has no main spec file %s.md — skipping.", specDirName, specDirName))
			continue
		}

		relSpecFilePath := filepath.ToSlash(filepath.Join("specs", specDirName, specDirName+".md"))

		spec, parseErr := parseSpec(relSpecFilePath, string(specFileRaw))
		if parseErr != nil {
			warnings = append(warnings, fmt.Sprintf("Skipping malformed spec file %s: %s", relSpecFilePath, parseErr.Error()))
		} else {
			if existing, dup := specIDsSeen[spec.ID]; dup {
				return nil, fmt.Errorf("duplicate spec id %q found in files: %s, %s", spec.ID, existing, relSpecFilePath)
			}
			specIDsSeen[spec.ID] = relSpecFilePath
			specs = append(specs, *spec)
		}

		// Check assertions directory.
		// A spec directory with no assertions/ directory is a normal spec that
		// nobody has broken into assertions yet. The spec is already in the
		// result above, and spekk status shows it as 0/0 complete, so there is
		// nothing to report. This once warned that the spec was "skipping",
		// which was false on both counts.
		assertDirInfo, statErr := os.Stat(assertionsDirPath)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil || !assertDirInfo.IsDir() {
			warnings = append(warnings, fmt.Sprintf("%s/assertions is not a directory — skipping.", specDirName))
			continue
		}

		assertionEntries, readAssertErr := os.ReadDir(assertionsDirPath)
		if readAssertErr != nil {
			warnings = append(warnings, fmt.Sprintf("cannot read assertions dir for %s: %s", specDirName, readAssertErr.Error()))
			continue
		}

		sort.Slice(assertionEntries, func(i, j int) bool {
			return assertionEntries[i].Name() < assertionEntries[j].Name()
		})

		assertionIDsSeen := make(map[string]string)

		for _, aEntry := range assertionEntries {
			if aEntry.IsDir() || !strings.HasSuffix(aEntry.Name(), ".md") {
				continue
			}

			aFileName := aEntry.Name()
			aFilePath := filepath.Join(assertionsDirPath, aFileName)
			relAFilePath := filepath.ToSlash(filepath.Join("specs", specDirName, "assertions", aFileName))

			aRaw, aReadErr := os.ReadFile(aFilePath)
			if aReadErr != nil {
				warnings = append(warnings, fmt.Sprintf("Skipping unreadable assertion file %s: %s", relAFilePath, aReadErr.Error()))
				continue
			}

			aContent := string(aRaw)

			// Silently skip files without frontmatter.
			if !hasFrontmatter(aContent) {
				continue
			}

			assertion, aParseErr := parseAssertion(relAFilePath, aContent)
			if aParseErr != nil {
				warnings = append(warnings, fmt.Sprintf("Skipping malformed assertion file %s: %s", relAFilePath, aParseErr.Error()))
				continue
			}

			if existing, dup := assertionIDsSeen[assertion.ID]; dup {
				warnings = append(warnings, fmt.Sprintf("Duplicate assertion id %q in spec %q: %s and %s — skipping second.",
					assertion.ID, specDirName, existing, aFileName))
				continue
			}
			assertionIDsSeen[assertion.ID] = aFileName

			assertions = append(assertions, *assertion)
		}
	}

	// Post-parse validation: assertion parent references existing specs.
	for _, a := range assertions {
		if _, exists := specIDsSeen[a.Parent]; !exists {
			return nil, fmt.Errorf("Parent spec %q not found for assertion %q", a.Parent, a.ID)
		}
	}

	// Post-parse validation: depends-on fields.
	assertionIDs := make(map[string]bool)
	for _, a := range assertions {
		assertionIDs[a.ID] = true
	}
	for _, a := range assertions {
		if a.DependsOn == "" {
			continue
		}
		if !kebabCasePattern.MatchString(a.DependsOn) {
			return nil, fmt.Errorf("Field 'depends-on' must be kebab-case (lowercase with hyphens) in %s\nFound: %q", a.File, a.DependsOn)
		}
		if a.DependsOn == a.ID {
			return nil, fmt.Errorf("Field 'depends-on' cannot reference itself in %s", a.File)
		}
		if !assertionIDs[a.DependsOn] {
			return nil, fmt.Errorf("Field 'depends-on' references non-existent assertion %q in %s", a.DependsOn, a.File)
		}
	}

	// Post-parse validation: circular dependency detection.
	if err := detectCircularDependencies(assertions); err != nil {
		return nil, err
	}

	// Derive parent spec statuses from child assertion statuses.
	for i := range specs {
		if specs[i].Status != "draft" {
			specs[i].Status = computeParentStatus(specs[i].ID, assertions)
		}
	}

	return &ParseResult{
		Specs:      specs,
		Assertions: assertions,
		Warnings:   warnings,
	}, nil
}

// WarningSummary renders Warnings as the one line a command shows instead of
// one line per skipped file. It returns "" when there is nothing to report, so
// a caller prints it unconditionally and stays silent on a clean tree.
//
// The detail lives in spekk validate, which reports each of these as a failure
// naming the file and the exact fault.
func (r *ParseResult) WarningSummary() string {
	if len(r.Warnings) == 0 {
		return ""
	}
	noun := "spec files"
	if len(r.Warnings) == 1 {
		noun = "spec file"
	}
	return fmt.Sprintf("Warning: %d %s skipped and missing from the queue. Run \"spekk validate\" for detail.",
		len(r.Warnings), noun)
}

// ParentStatusFromChildStatuses derives a spec's rolled-up status from its child
// assertions' statuses, using the same rules as the working-tree parser
// (computeParentStatus). It is exposed so callers that assemble a spec's children
// from outside the working tree — e.g. the cross-branch explorer synthesizing a
// foreign spec from assertions parsed out of a git ref — compute a status
// consistent with how local specs are rolled up.
func ParentStatusFromChildStatuses(statuses []string) string {
	children := make([]Assertion, len(statuses))
	for i, s := range statuses {
		children[i] = Assertion{Parent: "p", Status: s}
	}
	return computeParentStatus("p", children)
}

// computeParentStatus derives the status of a spec from its child assertions.
func computeParentStatus(parentID string, assertions []Assertion) string {
	var children []Assertion
	for _, a := range assertions {
		if a.Parent == parentID {
			children = append(children, a)
		}
	}

	if len(children) == 0 {
		return "not_started"
	}

	var active []Assertion
	for _, c := range children {
		if c.Status != "draft" {
			active = append(active, c)
		}
	}

	if len(active) == 0 {
		return "not_started"
	}

	for _, c := range active {
		if c.Status == "failed" {
			return "failed"
		}
	}

	allDone := true
	for _, c := range active {
		if c.Status != "done" {
			allDone = false
			break
		}
	}
	if allDone {
		return "done"
	}

	return "in_progress"
}

// detectCircularDependencies checks all assertions for dependency cycles.
// Returns an error with the cycle path if a circular dependency is found.
func detectCircularDependencies(assertions []Assertion) error {
	depMap := make(map[string]string) // id -> dependsOn
	for _, a := range assertions {
		if a.DependsOn != "" {
			depMap[a.ID] = a.DependsOn
		}
	}

	for _, a := range assertions {
		if a.DependsOn == "" {
			continue
		}

		visited := make(map[string]bool)
		var path []string
		current := a.ID

		for current != "" {
			if visited[current] {
				// Found a cycle — build the cycle path.
				path = append(path, current)
				cycleStart := -1
				for i, id := range path {
					if id == current {
						cycleStart = i
						break
					}
				}
				cycle := strings.Join(path[cycleStart:], " → ")
				return fmt.Errorf("Circular dependency detected:\n  %s\n\nBreak the cycle by removing or changing one of the dependencies.", cycle)
			}

			visited[current] = true
			path = append(path, current)
			current = depMap[current]
		}
	}
	return nil
}

// lockLifetime is how long a builder's claim on an assertion stands. A builder
// works one assertion at a time, so a claim older than this is almost always a
// session that died rather than one still running.
const lockLifetime = 2 * time.Hour

// IsLockStale reports whether a lockedBy value names a claim that no longer
// holds. A lock has the shape builder-{host}-{pid}-{unix-timestamp}, so the
// last hyphen-separated field dates it.
//
// An empty or undatable value is stale: a claim nobody can date is a claim
// nobody can trust.
func IsLockStale(lockedBy string) bool {
	if lockedBy == "" {
		return true
	}

	parts := strings.Split(lockedBy, "-")
	ts, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return true
	}

	return time.Since(time.Unix(ts, 0)) > lockLifetime
}

// FindOptions controls behaviour of FindNextAssertion.
type FindOptions struct {
	AssertionID   string
	SpecID        string
	AllBranches   bool
	CurrentBranch string
}

// FindNextAssertion selects the highest-priority incomplete assertion.
// It filters by branch (unless AllBranches is true), dependency satisfaction,
// and lock staleness.
func FindNextAssertion(assertions []Assertion, specs []Spec, opts FindOptions) *Assertion {
	if opts.AssertionID != "" {
		for i := range assertions {
			if assertions[i].ID == opts.AssertionID {
				return &assertions[i]
			}
		}
		return nil
	}

	specStatus := make(map[string]string)
	for _, s := range specs {
		specStatus[s.ID] = s.Status
	}

	assertionStatus := make(map[string]string)
	for _, a := range assertions {
		assertionStatus[a.ID] = a.Status
	}

	var candidates []Assertion
	for _, a := range assertions {
		if a.Status == "done" || a.Status == "draft" {
			continue
		}

		if specStatus[a.Parent] == "draft" {
			continue
		}

		if !opts.AllBranches && opts.CurrentBranch != "" {
			if a.Branch != "" && a.Branch != opts.CurrentBranch {
				continue
			}
		}

		if opts.SpecID != "" && a.Parent != opts.SpecID {
			continue
		}

		if a.DependsOn != "" && assertionStatus[a.DependsOn] != "done" {
			continue
		}

		if a.Status == "in_progress" && a.LockedBy != "" && !IsLockStale(a.LockedBy) {
			continue
		}

		candidates = append(candidates, a)
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority < candidates[j].Priority
		}
		if candidates[i].Created != candidates[j].Created {
			return candidates[i].Created < candidates[j].Created
		}
		return candidates[i].ID < candidates[j].ID
	})

	return &candidates[0]
}
