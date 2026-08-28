// Package show generates and opens the Spec Explorer HTML interface.
package show

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/crossbranch"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

//go:embed template.html
var templateHTML string

// showData is the top-level data structure injected into the template.
type showData struct {
	ProjectName string          `json:"projectName"`
	Specs       []showSpec      `json:"specs"`
	Assertions  []showAssertion `json:"assertions"`

	// CrossBranch reports whether cross-branch / merge-preview mode is active.
	// When false the cross-branch fields below and on each spec/assertion are
	// empty/omitted, and show output is byte-identical to the default path.
	CrossBranch bool `json:"crossBranch,omitempty"`
	// Degraded is true when conflict detection is unavailable (git < 2.38, or an
	// unparseable version) so the template can show the "classification-only,
	// conflicts unconfirmed" notice. Only meaningful when CrossBranch is true.
	Degraded bool `json:"degraded,omitempty"`
	// Branches is the set of comparison branch display names folded into the
	// view (the union of branches that contributed at least one state).
	Branches []string `json:"branches,omitempty"`
}

// crossBranchContribution is one (branch, state) contribution for a spec or
// assertion in cross-branch mode. A single item may carry several of these (N
// branches collapsed into one view). The shape is intentionally keyed by branch
// and not collapsed to a single pair so the future A-vs-B extension can add more
// pairwise contributions without a model rewrite.
type crossBranchContribution struct {
	Branch    string `json:"branch"`
	State     string `json:"state"` // incoming_add|incoming_mod|conflict|incoming_del
	Degraded  bool   `json:"degraded,omitempty"`
	OldStatus string `json:"oldStatus,omitempty"`
	NewStatus string `json:"newStatus,omitempty"`
}

// showSpec represents a spec for the explorer UI.
type showSpec struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	File     string `json:"file"`
	Content  string `json:"content"`
	Branch   string `json:"branch"`

	// Foreign is true when this spec exists only on other branches (an incoming
	// addition with no local working-tree file) and was synthesized from metadata
	// parsed out of a ref. The client uses it to hide the item when all of its
	// contributing branches are deselected.
	Foreign bool `json:"foreign,omitempty"`

	// CrossBranch holds the per-branch contributions touching this spec's parent
	// file (omitted entirely when not in cross-branch mode or when unchanged).
	CrossBranch []crossBranchContribution `json:"crossBranch,omitempty"`
	// CrossBranchSummary is the rolled-up headline state across this spec and all
	// its assertions, suitable for a single spec-level badge. The rollup picks the
	// "worst"/most-attention-worthy state by the precedence:
	//
	//	conflict > incoming_del > incoming_add > incoming_mod
	//
	// (a conflict anywhere dominates; otherwise an incoming deletion, then an
	// incoming addition, then a clean incoming modification). Empty when the spec
	// and its assertions have no cross-branch contributions.
	CrossBranchSummary string `json:"crossBranchSummary,omitempty"`
}

// showAssertion represents an assertion for the explorer UI.
type showAssertion struct {
	ID        string `json:"id"`
	Parent    string `json:"parent"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Priority  int    `json:"priority"`
	File      string `json:"file"`
	Content   string `json:"content"`
	Branch    string `json:"branch"`
	DependsOn string `json:"dependsOn,omitempty"`
	Created   string `json:"created"`

	// Foreign is true when this assertion exists only on other branches (an
	// incoming addition with no local working-tree file) and was synthesized from
	// metadata parsed out of a ref.
	Foreign bool `json:"foreign,omitempty"`

	// CrossBranch holds the per-branch contributions touching this assertion file
	// (omitted entirely when not in cross-branch mode or when unchanged).
	CrossBranch []crossBranchContribution `json:"crossBranch,omitempty"`
}

// Options configures how the Spec Explorer is generated.
type Options struct {
	// CrossBranch activates cross-branch / merge-preview mode. When false,
	// show behaves exactly as the current-working-tree-only default.
	CrossBranch bool
	// BranchFilter is an optional glob used to exclude noisy/stale branches
	// in cross-branch mode (e.g. "feat/*"). Empty means no filtering.
	BranchFilter string
}

// Run parses specs from specsDir, generates the Spec Explorer HTML, writes it
// to .spekk/index.html relative to the project root, and opens it in the
// default browser.
//
// When opts.CrossBranch is false, behavior is identical to the current
// working-tree-only default. When true, cross-branch / merge-preview mode
// classifies spec/assertion state across all branches (via applyCrossBranch) and
// folds the contributions into the rendered data.
func Run(specsDir string, opts Options) error {
	// 1. Parse specs
	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return fmt.Errorf("parsing specs: %w", err)
	}
	if summary := result.WarningSummary(); summary != "" {
		fmt.Fprintln(os.Stderr, summary)
	}

	if len(result.Specs) == 0 {
		return fmt.Errorf("no specifications found in %s", specsDir)
	}

	// 2. Build showData from parse result
	data := buildShowData(specsDir, result)

	// 2a. In cross-branch mode, classify changed spec/assertion files across all
	// branches and fold the contributions into the data. When off, this is
	// skipped entirely so the output is byte-identical to the default path.
	if opts.CrossBranch {
		if err := applyCrossBranch(&data, opts.BranchFilter); err != nil {
			return fmt.Errorf("classifying cross-branch state: %w", err)
		}
	}

	// 3. Marshal to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling data to JSON: %w", err)
	}

	// 4. Replace placeholder in template
	html := strings.Replace(templateHTML, "/*__SPEKK_DATA__*/", string(jsonBytes), 1)

	// 5. Determine output path (.spekk/index.html relative to project root)
	projectRoot := filepath.Dir(specsDir) // specsDir is <root>/specs
	spekkDir := filepath.Join(projectRoot, ".spekk")

	if err := os.MkdirAll(spekkDir, 0o755); err != nil {
		return fmt.Errorf("creating .spekk directory: %w", err)
	}

	outputPath := filepath.Join(spekkDir, "index.html")
	if err := os.WriteFile(outputPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outputPath, err)
	}

	fmt.Fprintf(os.Stderr, "Spec Explorer written to %s\n", outputPath)

	// 6. Open in browser
	if err := openBrowser(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser: %s\n", err)
		fmt.Fprintf(os.Stderr, "Open the file manually: %s\n", outputPath)
	}

	return nil
}

// buildShowData converts the parser result into the JSON structure expected by
// the template.
func buildShowData(specsDir string, result *parser.ParseResult) showData {
	projectName := filepath.Base(filepath.Dir(specsDir))

	showSpecs := make([]showSpec, len(result.Specs))
	for i, s := range result.Specs {
		showSpecs[i] = showSpec{
			ID:       s.ID,
			Title:    s.Title,
			Status:   s.Status,
			Priority: s.Priority,
			File:     s.File,
			Content:  s.Content,
			Branch:   s.Branch,
		}
	}

	showAssertions := make([]showAssertion, len(result.Assertions))
	for i, a := range result.Assertions {
		showAssertions[i] = showAssertion{
			ID:        a.ID,
			Parent:    a.Parent,
			Title:     a.Title,
			Status:    a.Status,
			Priority:  a.Priority,
			File:      a.File,
			Content:   a.Content,
			Branch:    a.Branch,
			DependsOn: a.DependsOn,
			Created:   a.Created,
		}
	}

	return showData{
		ProjectName: projectName,
		Specs:       showSpecs,
		Assertions:  showAssertions,
	}
}

// rollupPrecedence ranks cross-branch states for the spec-level rollup badge.
// Higher wins. The rule (documented on showSpec.CrossBranchSummary):
//
//	conflict > incoming_del > incoming_add > incoming_mod
var rollupPrecedence = map[string]int{
	string(crossbranch.StateIncomingMod): 1,
	string(crossbranch.StateIncomingAdd): 2,
	string(crossbranch.StateIncomingDel): 3,
	string(crossbranch.StateConflict):    4,
}

// worseState returns whichever of a or b ranks higher by rollupPrecedence.
// Empty strings rank below everything.
func worseState(a, b string) string {
	if rollupPrecedence[b] > rollupPrecedence[a] {
		return b
	}
	return a
}

// applyCrossBranch classifies cross-branch state for the given branch filter and
// folds the contributions into data. It is the entry point for the non-watch
// path (show.Run); watch mode reuses foldCrossBranch directly with a cached
// classification keyed on git ref state (see RunWatch).
func applyCrossBranch(data *showData, filter string) error {
	states, supported, err := crossbranch.Classify(filter)
	if err != nil {
		return err
	}
	foldCrossBranch(data, states, supported)
	return nil
}

// foldCrossBranch folds an already-computed classification into data: attaching
// per-item contribution lists, synthesizing placeholder entries for incoming
// additions (foreign specs/assertions that have no local file), computing the
// spec-level rollup summary, and recording mode metadata (active, degraded,
// compared branches). Splitting this from classification lets watch mode cache
// the expensive Classify/SupportsMergeTree results while still re-folding into a
// freshly parsed data set on every render.
func foldCrossBranch(data *showData, states []crossbranch.FileState, supported bool) {
	data.CrossBranch = true
	data.Degraded = !supported

	// Phase 1: group contributions by repo-relative path. The parser stores File
	// as a slash-form path under "specs/", which is exactly the form
	// crossbranch.FileState.Path uses, so they match directly after normalize.
	// Grouping first (rather than mutating slice elements as we go) avoids holding
	// element pointers across the appends that synthesize foreign entries.
	byPath := map[string][]crossBranchContribution{}
	branchSet := map[string]struct{}{}
	// foreignMetaByPath holds the metadata a foreign (incoming-add) path should be
	// synthesized from. states arrives sorted by (Path, Branch), so the first Meta
	// seen for a path is the alphabetically-first contributing branch's — the
	// chosen source when a foreign file exists on several branches.
	foreignMetaByPath := map[string]*crossbranch.FileMeta{}
	for _, fs := range states {
		branchSet[fs.Branch] = struct{}{}
		path := normalizePath(fs.Path)
		byPath[path] = append(byPath[path], crossBranchContribution{
			Branch:    fs.Branch,
			State:     string(fs.State),
			Degraded:  fs.Degraded,
			OldStatus: fs.OldStatus,
			NewStatus: fs.NewStatus,
		})
		if fs.Meta != nil {
			if _, ok := foreignMetaByPath[path]; !ok {
				foreignMetaByPath[path] = fs.Meta
			}
		}
	}

	// Phase 2: attach contributions to existing items, tracking which paths were
	// matched so the rest (incoming additions with no local file) can be
	// synthesized afterward.
	matched := map[string]bool{}
	for i := range data.Specs {
		if c, ok := byPath[normalizePath(data.Specs[i].File)]; ok {
			data.Specs[i].CrossBranch = c
			matched[normalizePath(data.Specs[i].File)] = true
		}
	}
	for i := range data.Assertions {
		if c, ok := byPath[normalizePath(data.Assertions[i].File)]; ok {
			data.Assertions[i].CrossBranch = c
			matched[normalizePath(data.Assertions[i].File)] = true
		}
	}

	// Phase 3: synthesize entries for any contribution path with no local file.
	// These are foreign items: usually incoming additions, but also modify/delete
	// conflicts where ours deleted a file another branch modified (so it has no
	// local entry yet still contributes a conflict). In both cases the metadata was
	// parsed from the contributing branch during classification (foreignMetaByPath).
	// Synthesis happens after Phase 2 so the appends never invalidate live element
	// pointers.
	for path, contribs := range byPath {
		if matched[path] {
			continue
		}
		if isAssertionPath(path) {
			a := synthesizeAssertion(path, foreignMetaByPath[path])
			a.CrossBranch = contribs
			data.Assertions = append(data.Assertions, a)
		} else {
			s := synthesizeSpec(path, foreignMetaByPath[path])
			s.CrossBranch = contribs
			data.Specs = append(data.Specs, s)
		}
	}

	// A foreign spec's status is derived from its (foreign) assertions, exactly as
	// a local spec's is, since a spec parent file does not store status. Compute it
	// from the synthesized foreign assertions grouped by spec directory, so a
	// foreign spec renders a real status badge instead of an empty one.
	foreignChildStatuses := map[string][]string{}
	for i := range data.Assertions {
		if data.Assertions[i].Foreign {
			k := specDirKey(data.Assertions[i].File)
			foreignChildStatuses[k] = append(foreignChildStatuses[k], data.Assertions[i].Status)
		}
	}
	for i := range data.Specs {
		if data.Specs[i].Foreign {
			data.Specs[i].Status = parser.ParentStatusFromChildStatuses(foreignChildStatuses[specDirKey(data.Specs[i].File)])
		}
	}

	// Roll up each spec's headline state across its own contributions and those
	// of its assertions. Key by the spec DIRECTORY derived from file paths, not by
	// frontmatter id: a synthesized foreign assertion only knows its path, and the
	// parser does not require a spec's directory name to equal its frontmatter id.
	// Keying by directory makes real and synthesized entries align regardless.
	rollup := map[string]string{} // spec dir (e.g. "specs/foo") -> worst state
	for i := range data.Specs {
		s := &data.Specs[i]
		for _, c := range s.CrossBranch {
			k := specDirKey(s.File)
			rollup[k] = worseState(rollup[k], c.State)
		}
	}
	for i := range data.Assertions {
		a := &data.Assertions[i]
		for _, c := range a.CrossBranch {
			k := specDirKey(a.File)
			rollup[k] = worseState(rollup[k], c.State)
		}
	}
	for i := range data.Specs {
		data.Specs[i].CrossBranchSummary = rollup[specDirKey(data.Specs[i].File)]
	}

	branches := make([]string, 0, len(branchSet))
	for b := range branchSet {
		branches = append(branches, b)
	}
	sort.Strings(branches)
	data.Branches = branches
}

// normalizePath makes a file path comparable across the parser (which stores
// slash-form repo-relative paths under "specs/") and crossbranch.FileState.Path
// (likewise repo-relative slash-form under "specs/"). It coerces to slash form
// and strips any leading "./" so the two domains line up.
func normalizePath(p string) string {
	p = filepath.ToSlash(p)
	return strings.TrimPrefix(p, "./")
}

// isAssertionPath reports whether a repo-relative spec path is an assertion file
// (lives under an assertions/ directory) rather than a spec parent file.
func isAssertionPath(path string) bool {
	return strings.Contains(path, "/assertions/")
}

// idFromPath derives a stable id from a spec/assertion file path: the file's
// base name without its .md extension (e.g. "specs/foo/assertions/bar.md" ->
// "bar"). Used for synthesized foreign entries where no parsed id is available.
func idFromPath(path string) string {
	base := path[strings.LastIndex(path, "/")+1:]
	return strings.TrimSuffix(base, ".md")
}

// parentFromAssertionPath derives the owning spec id for an assertion file path
// of the form "specs/<spec-id>/assertions/<name>.md".
func parentFromAssertionPath(path string) string {
	const marker = "/assertions/"
	idx := strings.Index(path, marker)
	if idx < 0 {
		return ""
	}
	head := path[:idx] // "specs/<spec-id>"
	return head[strings.LastIndex(head, "/")+1:]
}

// specDirKey returns the spec-directory portion of a spec or assertion file path
// (e.g. "specs/foo" for both "specs/foo/foo.md" and
// "specs/foo/assertions/bar.md"). It is the stable rollup key tying an assertion
// to its owning spec by location, independent of frontmatter ids.
func specDirKey(path string) string {
	p := normalizePath(path)
	if idx := strings.Index(p, "/assertions/"); idx >= 0 {
		return p[:idx]
	}
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		return p[:idx]
	}
	return p
}

// specDirName returns the final path segment of specDirKey — the spec directory
// name (e.g. "foo" for both "specs/foo/foo.md" and "specs/foo/assertions/bar.md").
// This is the stable spec id for synthesized foreign entries, aligned with the
// rollup key and with parentFromAssertionPath.
func specDirName(path string) string {
	dir := specDirKey(path)
	return dir[strings.LastIndex(dir, "/")+1:]
}

// synthesizeSpec builds a showSpec for a foreign spec that exists only on another
// branch (incoming addition) and therefore has no local file to walk. The id is
// derived from the spec DIRECTORY name (not the file basename) so it matches both
// the rollup key (specDirKey) and the parent id that synthesizeAssertion derives
// via parentFromAssertionPath — keeping a foreign spec and its foreign assertions
// linked in the tree even when the parser's frontmatter id would differ. When
// meta is available (parsed from the contributing branch) the real title, priority
// and content are used; Status is computed by the caller from the foreign
// assertions. meta is nil only when the foreign file could not be parsed, in which
// case a path-derived placeholder is used.
func synthesizeSpec(path string, meta *crossbranch.FileMeta) showSpec {
	id := specDirName(path)
	s := showSpec{
		ID:      id,
		Title:   id,
		File:    path,
		Foreign: true,
	}
	if meta != nil {
		if meta.Title != "" {
			s.Title = meta.Title
		}
		s.Priority = meta.Priority
		s.Content = meta.Content
		s.Branch = meta.Branch
	}
	return s
}

// synthesizeAssertion builds a showAssertion for a foreign assertion that exists
// only on another branch (incoming addition). When meta is available the real
// title, status, priority, content and branch are used so the item renders with a
// proper status/priority badge rather than a blank placeholder.
func synthesizeAssertion(path string, meta *crossbranch.FileMeta) showAssertion {
	id := idFromPath(path)
	a := showAssertion{
		ID:      id,
		Parent:  parentFromAssertionPath(path),
		Title:   id,
		File:    path,
		Foreign: true,
	}
	if meta != nil {
		if meta.Title != "" {
			a.Title = meta.Title
		}
		a.Status = meta.Status
		a.Priority = meta.Priority
		a.Content = meta.Content
		a.Branch = meta.Branch
		// Prefer the frontmatter parent over the path-derived one: an assertion
		// may live under a spec directory whose name differs from the frontmatter
		// parent id, and the explicit frontmatter value is authoritative.
		if meta.Parent != "" {
			a.Parent = meta.Parent
		}
	}
	return a
}

// openBrowser opens the given file path in the default browser.
func openBrowser(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	u, err := url.Parse("file://" + absPath)
	if err != nil {
		return fmt.Errorf("failed to parse file URL: %w", err)
	}
	urlStr := u.String()

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", urlStr).Start()
	case "linux":
		return exec.Command("xdg-open", urlStr).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", urlStr).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
