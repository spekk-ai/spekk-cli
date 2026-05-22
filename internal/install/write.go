package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Destination returns the absolute path where a skill should be written for
// the given scope. The returned path always ends in `<skill>.md` so callers
// can write to it directly.
func Destination(cwd, home string, scope Scope, agent, skill string) (string, error) {
	if agent == "" {
		return "", errors.New("agent is required")
	}
	if skill == "" {
		return "", errors.New("skill is required")
	}

	var root string
	switch scope {
	case ScopeGlobal:
		if home == "" {
			return "", errors.New("home directory is unknown; cannot resolve --global scope")
		}
		root = home
	default:
		if cwd == "" {
			return "", errors.New("working directory is unknown; cannot resolve --local scope")
		}
		root = cwd
	}

	return filepath.Join(root, ".spekk", "skills", agent, skill+".md"), nil
}

// WriteSkill writes body to dest, creating any missing parent directories
// with mode 0755 and the file itself with mode 0644. When force is false
// and the destination already exists, returns an error naming the path and
// suggesting --force.
func WriteSkill(dest string, body []byte, force bool) error {
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("file already exists at %s (pass --force to overwrite)", dest)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create directory for %s: %w", dest, err)
	}

	if err := os.WriteFile(dest, body, 0644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}

// InstallRequest captures everything PerformInstall needs to fetch and write
// a skill. Tests inject FetchFn/FetchURL stubs; production callers use
// FetchSkill and FetchURL from this package.
type InstallRequest struct {
	Cwd     string
	HomeDir string
	Scope   Scope
	Agent   string
	Skill   string
	Source  string
	Force   bool

	FetchFn  func(agent, skill string) ([]byte, error)
	FetchURL func(url string) ([]byte, error)
}

// PerformInstall fetches the skill body, writes it to the scope-resolved
// destination, and returns the one-line confirmation message that the CLI
// should print to stdout.
func PerformInstall(req InstallRequest) (string, error) {
	dest, err := Destination(req.Cwd, req.HomeDir, req.Scope, req.Agent, req.Skill)
	if err != nil {
		return "", err
	}

	var body []byte
	if req.Source != "" {
		body, err = req.FetchURL(req.Source)
	} else {
		body, err = req.FetchFn(req.Agent, req.Skill)
	}
	if err != nil {
		return "", err
	}

	if err := WriteSkill(dest, body, req.Force); err != nil {
		return "", err
	}

	return fmt.Sprintf("installed %s/%s → %s", req.Agent, req.Skill, dest), nil
}
