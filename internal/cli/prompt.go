package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const promptSeparator = "\n\n---\n\n"

// validAgents lists the supported agent names.
var validAgents = []string{"coach", "builder", "observer"}

// PromptResolver resolves layered agent prompts.
type PromptResolver struct {
	HomeDir    string
	Cwd        string
	InstallDir string
}

// NewPromptResolver creates a resolver with default paths.
func NewPromptResolver(installDir string) *PromptResolver {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return &PromptResolver{
		HomeDir:    home,
		Cwd:        cwd,
		InstallDir: installDir,
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

// basePromptPath returns the package base prompt path for an agent.
func (r *PromptResolver) basePromptPath(agent string) string {
	return filepath.Join(r.InstallDir, "specs", agent+"-agent", agent+".prompt.md")
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
//
// Base (first match wins):
//  1. Local override: .spekk/{agent}.prompt.override.md
//  2. Global override: ~/.spekk/{agent}.prompt.override.md
//  3. Package base: specs/{agent}-agent/{agent}.prompt.md
//
// Extensions (appended in order):
//  1. Global extend: ~/.spekk/{agent}.prompt.md
//  2. Local extend: .spekk/{agent}.prompt.md
func (r *PromptResolver) GetPromptContent(agent string) (string, error) {
	if !isValidAgent(agent) {
		return "", fmt.Errorf("Unknown agent: %s", agent)
	}

	globalDir := filepath.Join(r.HomeDir, ".spekk")
	localDir := filepath.Join(r.Cwd, ".spekk")

	// Step 1: Determine base prompt
	var base string

	localOverridePath := filepath.Join(localDir, agent+".prompt.override.md")
	if content, ok := readIfExists(localOverridePath); ok {
		base = content
	} else {
		globalOverridePath := filepath.Join(globalDir, agent+".prompt.override.md")
		if content, ok := readIfExists(globalOverridePath); ok {
			base = content
		} else {
			basePath := r.basePromptPath(agent)
			content, ok := readIfExists(basePath)
			if !ok {
				return "", fmt.Errorf("Prompt file not found: %s", basePath)
			}
			base = content
		}
	}

	// Step 2: Collect layers
	layers := []string{base}

	// Global extend
	globalExtendPath := filepath.Join(globalDir, agent+".prompt.md")
	if content, ok := readIfExists(globalExtendPath); ok {
		layers = append(layers, content)
	}

	// Local extend
	localExtendPath := filepath.Join(localDir, agent+".prompt.md")
	if content, ok := readIfExists(localExtendPath); ok {
		layers = append(layers, content)
	}

	// Step 3: Concatenate
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
Spekk installation: %s

Here is your prompt:

%s`, displayName, workingDir, r.InstallDir, promptContent), nil
}
