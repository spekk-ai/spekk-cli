package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Uninstall deletes the skill file at the scope-resolved destination for
// (agent, skill). Returns the absolute path that was removed (for the
// confirmation message) on success, or an error.
//
// Errors:
//   - The file does not exist: returns a "not installed: <path>" error so
//     callers can exit non-zero and scripts can detect the case.
//   - The resolved destination escapes the scope's `.spekk/skills/<agent>/`
//     directory: returns an error and the filesystem is not touched. This
//     is a defense against a future caller passing an agent/skill name
//     containing path-traversal segments.
func Uninstall(cwd, home string, scope Scope, agent, skill string) (string, error) {
	dest, err := Destination(cwd, home, scope, agent, skill)
	if err != nil {
		return "", err
	}

	scopeRoot, err := scopeDir(cwd, home, scope, agent)
	if err != nil {
		return "", err
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolve dest path: %w", err)
	}
	absScope, err := filepath.Abs(scopeRoot)
	if err != nil {
		return "", fmt.Errorf("resolve scope path: %w", err)
	}

	if !strings.HasPrefix(absDest+string(os.PathSeparator), absScope+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing to uninstall outside scope directory %s", absScope)
	}

	if _, err := os.Stat(absDest); os.IsNotExist(err) {
		return "", fmt.Errorf("not installed: %s", absDest)
	} else if err != nil {
		return "", fmt.Errorf("stat %s: %w", absDest, err)
	}

	if err := os.Remove(absDest); err != nil {
		return "", fmt.Errorf("remove %s: %w", absDest, err)
	}
	return absDest, nil
}

// scopeDir returns the directory that bounds an uninstall for the given
// scope and agent — `<root>/.spekk/skills/<agent>/`. Uninstall must never
// delete files outside this directory.
func scopeDir(cwd, home string, scope Scope, agent string) (string, error) {
	if agent == "" {
		return "", fmt.Errorf("agent is required")
	}
	var root string
	switch scope {
	case ScopeGlobal:
		if home == "" {
			return "", fmt.Errorf("home directory is unknown; cannot resolve --global scope")
		}
		root = home
	default:
		if cwd == "" {
			return "", fmt.Errorf("working directory is unknown; cannot resolve --local scope")
		}
		root = cwd
	}
	return filepath.Join(root, ".spekk", "skills", agent), nil
}
