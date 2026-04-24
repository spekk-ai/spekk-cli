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
}

// ParseResult holds the full result of parsing a specs directory.
type ParseResult struct {
	Specs      []Spec
	Assertions []Assertion
}

// frontmatter holds raw parsed YAML frontmatter fields.
type frontmatter struct {
	fields map[string]string
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

	fm := &frontmatter{fields: make(map[string]string)}

	for _, line := range yamlLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Skip YAML array items (we only handle scalar values).
		if strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		// Strip surrounding quotes if present.
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		fm.fields[key] = value
	}

	return fm, markdownContent, nil
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

// validBranchPattern matches valid git branch name characters.
var validBranchPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9/_-]*$`)

// standardBranchPattern matches commonly used branch naming patterns.
var standardBranchPattern = regexp.MustCompile(`^(main|master|develop|feature/|bugfix/|hotfix/|release/)`)

// validateBranch validates a branch field value. Returns an error for invalid branches
// and prints a warning to stderr for non-standard patterns.
func validateBranch(branch, filePath string) error {
	if branch == "" {
		return nil
	}
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("Field 'branch' cannot start or end with '/' in %s", filePath)
	}
	if !validBranchPattern.MatchString(branch) {
		return fmt.Errorf("Field 'branch' contains invalid characters in %s\nFound: %q\nGit branch names can only contain letters, numbers, slashes, hyphens, and underscores.", filePath, branch)
	}
	if !standardBranchPattern.MatchString(branch) {
		fmt.Fprintf(os.Stderr, "Warning: Field 'branch' uses non-standard pattern in %s\nFound: %q\nConsider using standard patterns: main, feature/<name>, bugfix/<name>, hotfix/<name>\n", filePath, branch)
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
		Content:  content,
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
		Content:   content,
	}, nil
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
			fmt.Fprintf(os.Stderr, "Warning: Spec %s/ has no main spec file %s.md — skipping.\n", specDirName, specDirName)
			continue
		}

		relSpecFilePath := filepath.ToSlash(filepath.Join("specs", specDirName, specDirName+".md"))

		spec, parseErr := parseSpec(relSpecFilePath, string(specFileRaw))
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Skipping malformed spec file %s: %s\n", relSpecFilePath, parseErr.Error())
		} else {
			if existing, dup := specIDsSeen[spec.ID]; dup {
				return nil, fmt.Errorf("duplicate spec id %q found in files: %s, %s", spec.ID, existing, relSpecFilePath)
			}
			specIDsSeen[spec.ID] = relSpecFilePath
			specs = append(specs, *spec)
		}

		// Check assertions directory.
		assertDirInfo, statErr := os.Stat(assertionsDirPath)
		if os.IsNotExist(statErr) {
			fmt.Fprintf(os.Stderr, "Warning: Spec %s/ has no assertions/ directory — skipping.\n", specDirName)
			continue
		}
		if statErr != nil || !assertDirInfo.IsDir() {
			fmt.Fprintf(os.Stderr, "Warning: %s/assertions is not a directory — skipping.\n", specDirName)
			continue
		}

		assertionEntries, readAssertErr := os.ReadDir(assertionsDirPath)
		if readAssertErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read assertions dir for %s: %s\n", specDirName, readAssertErr.Error())
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
				fmt.Fprintf(os.Stderr, "Warning: Skipping unreadable assertion file %s: %s\n", relAFilePath, aReadErr.Error())
				continue
			}

			aContent := string(aRaw)

			// Silently skip files without frontmatter.
			if !hasFrontmatter(aContent) {
				continue
			}

			assertion, aParseErr := parseAssertion(relAFilePath, aContent)
			if aParseErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Skipping malformed assertion file %s: %s\n", relAFilePath, aParseErr.Error())
				continue
			}

			if existing, dup := assertionIDsSeen[assertion.ID]; dup {
				fmt.Fprintf(os.Stderr, "Warning: Duplicate assertion id %q in spec %q: %s and %s — skipping second.\n",
					assertion.ID, specDirName, existing, aFileName)
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
	}, nil
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

// IsLockStale reports whether a lockedBy string represents a stale (>2 hour old) lock.
// Returns true for empty or malformed lockedBy values.
func IsLockStale(lockedBy string) bool {
	if lockedBy == "" {
		return true
	}

	parts := strings.Split(lockedBy, "-")
	if len(parts) == 0 {
		return true
	}

	tsStr := parts[len(parts)-1]
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return true
	}

	return time.Now().Unix()-ts > 7200
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
