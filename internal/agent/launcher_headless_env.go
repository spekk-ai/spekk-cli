package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// headlessChildEnv returns the environment for the headless Claude process
// with this binary's own directory prepended to PATH. Cron and systemd
// environments often lack the spekk install directory (~/.local/bin), and
// agent prompts run bare `spekk` commands inside the spawned session — without
// this, those calls fail with "command not found" while the outer absolute-path
// invocation works, which makes scheduled runs die silently.
func headlessChildEnv() []string {
	env := os.Environ()
	exe, err := os.Executable()
	if err != nil {
		return env
	}
	dir := filepath.Dir(exe)

	for i, kv := range env {
		eq := strings.Index(kv, "=")
		if eq < 0 {
			continue
		}
		if !strings.EqualFold(kv[:eq], "PATH") {
			continue
		}
		val := kv[eq+1:]
		for _, p := range filepath.SplitList(val) {
			if p == dir {
				return env
			}
		}
		env[i] = kv[:eq] + "=" + dir + string(os.PathListSeparator) + val
		return env
	}
	return append(env, "PATH="+dir)
}
