package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const promptSeparator = "\n\n---\n\n"

// DefaultEmbeddedFS is the embedded filesystem containing built-in agent prompts.
// Set this from main() before any prompt resolution occurs.
var DefaultEmbeddedFS fs.FS

// validAgents lists the supported agent names.
var validAgents = []string{"coach", "builder", "observer"}

// PromptResolver resolves layered agent prompts.
//
// Resolution order:
//
//	Base (first match wins):
//	  1. Local override:  .spekk/{agent}.prompt.override.md
//	  2. Global override: ~/.spekk/{agent}.prompt.override.md
//	  3. Embedded base:   built into the binary at compile time
//
//	Extensions (appended in order after base):
//	  1. Global extend: ~/.spekk/{agent}.prompt.md
//	  2. Local extend:  .spekk/{agent}.prompt.md
type PromptResolver struct {
	HomeDir    string
	Cwd        string
	EmbeddedFS fs.FS // override for testing; falls back to DefaultEmbeddedFS
}

// NewPromptResolver creates a resolver with default paths.
func NewPromptResolver() *PromptResolver {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return &PromptResolver{
		HomeDir: home,
		Cwd:     cwd,
	}
}

// isValidAgent checks if the agent name is recognized.
func isValidAgent(name string) bool {
	for _, a := range validAgents {
		if a == name {
			return true
		}
	}
	return false
}

// embeddedFS returns the embedded filesystem to use, preferring the
// instance field over the package-level default.
func (r *PromptResolver) embeddedFS() fs.FS {
	if r.EmbeddedFS != nil {
		return r.EmbeddedFS
	}
	return DefaultEmbeddedFS
}

// readBasePrompt reads the built-in base prompt from the embedded filesystem.
func (r *PromptResolver) readBasePrompt(agent string) (string, error) {
	efs := r.embeddedFS()
	if efs == nil {
		return "", fmt.Errorf("no embedded prompts available for agent %q", agent)
	}
	path := "specs/" + agent + "-agent/" + agent + ".prompt.md"
	data, err := fs.ReadFile(efs, path)
	if err != nil {
		return "", fmt.Errorf("embedded prompt not found for agent %q", agent)
	}
	return string(data), nil
}

// readIfExists reads a file if it exists, returning empty string and false if not.
func readIfExists(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// GetPromptContent resolves the full prompt for an agent using layered resolution.
func (r *PromptResolver) GetPromptContent(agent string) (string, error) {
	if !isValidAgent(agent) {
		return "", fmt.Errorf("Unknown agent: %s", agent)
	}

	globalDir := filepath.Join(r.HomeDir, ".spekk")
	localDir := filepath.Join(r.Cwd, ".spekk")

	// Step 1: Determine base prompt (first match wins)
	var base string

	localOverridePath := filepath.Join(localDir, agent+".prompt.override.md")
	if content, ok := readIfExists(localOverridePath); ok {
		base = content
	} else if content, ok := readIfExists(filepath.Join(globalDir, agent+".prompt.override.md")); ok {
		base = content
	} else {
		content, err := r.readBasePrompt(agent)
		if err != nil {
			return "", err
		}
		base = content
	}

	// Step 2: Append extensions
	layers := []string{base}

	// Global extend
	if content, ok := readIfExists(filepath.Join(globalDir, agent+".prompt.md")); ok {
		layers = append(layers, content)
	}

	// Local extend
	if content, ok := readIfExists(filepath.Join(localDir, agent+".prompt.md")); ok {
		layers = append(layers, content)
	}

	return strings.Join(layers, promptSeparator), nil
}

// CreateActivationMessage builds the full activation message for an agent.
func (r *PromptResolver) CreateActivationMessage(agent string) (string, error) {
	promptContent, err := r.GetPromptContent(agent)
	if err != nil {
		return "", err
	}

	displayName := strings.ToUpper(agent[:1]) + agent[1:]
	workingDir, _ := os.Getwd()

	return fmt.Sprintf(`You are the %s Agent - read the prompt and follow the instructions exactly.

Working directory: %s

Here is your prompt:

%s`, displayName, workingDir, promptContent), nil
}
