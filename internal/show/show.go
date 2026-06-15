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
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

//go:embed template.html
var templateHTML string

// showData is the top-level data structure injected into the template.
type showData struct {
	ProjectName string          `json:"projectName"`
	Specs       []showSpec      `json:"specs"`
	Assertions  []showAssertion `json:"assertions"`
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
// working-tree-only default. When true, cross-branch / merge-preview mode is
// requested; the real diffing is wired in by later work, so for now it is a
// no-op placeholder beyond a log line.
func Run(specsDir string, opts Options) error {
	if opts.CrossBranch {
		// Placeholder: cross-branch diffing is implemented by later waves.
		// For now this falls through to the standard current-tree behavior so
		// the build stays green and existing behavior is unchanged.
		fmt.Fprintln(os.Stderr, "cross-branch mode requested (not yet implemented)")
	}

	// 1. Parse specs
	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return fmt.Errorf("parsing specs: %w", err)
	}

	if len(result.Specs) == 0 {
		return fmt.Errorf("no specifications found in %s", specsDir)
	}

	// 2. Build showData from parse result
	data := buildShowData(specsDir, result)

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
