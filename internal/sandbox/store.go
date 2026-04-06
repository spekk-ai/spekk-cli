package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SandboxMeta holds local metadata for a sandbox.
type SandboxMeta struct {
	DropletID int    `json:"dropletId"`
	IP        string `json:"ip"`
	Region    string `json:"region"`
	Size      string `json:"size"`
	CreatedAt string `json:"createdAt"`
	Status    string `json:"status"`
	Project   string `json:"project,omitempty"`
}

// sandboxesFile returns the path to the sandboxes metadata file.
// Overridable in tests.
var sandboxesFile = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".spekk", "sandboxes.json")
}

// LoadSandboxes reads all sandbox metadata from ~/.spekk/sandboxes.json.
func LoadSandboxes() (map[string]*SandboxMeta, error) {
	data, err := os.ReadFile(sandboxesFile())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*SandboxMeta), nil
		}
		return nil, err
	}
	var result map[string]*SandboxMeta
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing sandboxes.json: %w", err)
	}
	if result == nil {
		result = make(map[string]*SandboxMeta)
	}
	return result, nil
}

// GetSandbox returns metadata for a named sandbox, or nil if not found.
func GetSandbox(name string) (*SandboxMeta, error) {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return nil, err
	}
	return sandboxes[name], nil
}

// SaveSandbox writes or updates metadata for a named sandbox.
func SaveSandbox(name string, meta *SandboxMeta) error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	sandboxes[name] = meta

	return writeSandboxes(sandboxes)
}

// RemoveSandbox deletes a sandbox from the metadata store.
func RemoveSandbox(name string) error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	delete(sandboxes, name)

	return writeSandboxes(sandboxes)
}

func writeSandboxes(sandboxes map[string]*SandboxMeta) error {
	dir := filepath.Dir(sandboxesFile())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating ~/.spekk: %w", err)
	}
	data, err := json.MarshalIndent(sandboxes, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(sandboxesFile(), data, 0o644)
}
