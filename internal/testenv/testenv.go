// Package testenv holds helpers that keep tests away from the developer's
// own machine state: the real config directory and the global git config.
// Only tests import it.
package testenv

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Snapshot records every file under dir with its size and modification time.
// A missing dir gives an empty snapshot.
func Snapshot(dir string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out[path] = info.ModTime().Format(time.RFC3339Nano) + " " + strconv.FormatInt(info.Size(), 10)
		return nil
	})
	return out
}

// GuardDir snapshots dir now and returns a function that reports whether the
// files under it changed since. Call it before a test run changes any
// environment variable, so dir is the developer's real directory.
func GuardDir(dir string) (changed func() []string) {
	before := Snapshot(dir)
	return func() []string {
		after := Snapshot(dir)
		var diff []string
		for path, v := range after {
			if before[path] != v {
				diff = append(diff, path)
			}
		}
		for path := range before {
			if _, ok := after[path]; !ok {
				diff = append(diff, path)
			}
		}
		return diff
	}
}

// RunGuarded runs the package tests (m.Run) and fails the run when any file
// under the real user config directory's spekk directory changed. Use it as
// the body of TestMain.
func RunGuarded(run func() int) int {
	dir, err := os.UserConfigDir()
	if err != nil {
		return run()
	}
	changed := GuardDir(filepath.Join(dir, "spekk"))
	code := run()
	if diff := changed(); len(diff) > 0 {
		os.Stderr.WriteString("tests changed files under the real user config directory:\n")
		for _, p := range diff {
			os.Stderr.WriteString("  " + p + "\n")
		}
		return 1
	}
	return code
}
