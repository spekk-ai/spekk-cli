package config

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/spekk-ai/spekk-cli/internal/fsutil"
)

var (
	once      sync.Once
	cachedDir string
	cachedErr error
)

// GlobalConfigDir returns the global spekk configuration directory, following
// the XDG Base Directory Specification:
//
//	$XDG_CONFIG_HOME/spekk  (if XDG_CONFIG_HOME is set and non-empty)
//	~/.config/spekk          (otherwise)
//
// On the first call per process, if the legacy ~/.spekk directory exists and
// the new path does not, the directory is migrated. When stdin is a terminal
// the user is asked to press Enter first; in non-interactive contexts (pipes,
// shims, servers) the migration happens silently with a notice on stderr.
func GlobalConfigDir() (string, error) {
	once.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			cachedErr = fmt.Errorf("getting home dir: %w", err)
			return
		}
		cachedDir, cachedErr = resolveGlobalConfigDir(home, os.Stderr, os.Stdin, stdinIsTTY())
	})
	return cachedDir, cachedErr
}

// DefaultDir returns the global config path without triggering migration.
// Use only as a fallback when GlobalConfigDir() has failed.
func DefaultDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "spekk")
}

// stdinIsTTY reports whether stdin looks like a terminal. Pipes and file
// redirects are correctly detected as non-interactive; /dev/null is a char
// device so it passes this check, but reads return EOF immediately so the
// migration prompt cannot block there.
func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func resolveGlobalConfigDir(home string, out io.Writer, in io.Reader, interactive bool) (string, error) {
	xdgBase := os.Getenv("XDG_CONFIG_HOME")
	if xdgBase == "" {
		xdgBase = filepath.Join(home, ".config")
	}
	newDir := filepath.Join(xdgBase, "spekk")
	oldDir := filepath.Join(home, ".spekk")
	if err := maybeMigrate(oldDir, newDir, out, in, interactive); err != nil {
		return "", err
	}
	return newDir, nil
}

func maybeMigrate(oldDir, newDir string, out io.Writer, in io.Reader, interactive bool) error {
	if !fsutil.DirExists(oldDir) || fsutil.DirExists(newDir) {
		return nil
	}
	fmt.Fprintf(out, "\nspekk: config directory has moved.\n")
	fmt.Fprintf(out, "  from: %s\n", oldDir)
	fmt.Fprintf(out, "    to: %s\n\n", newDir)
	if interactive {
		fmt.Fprint(out, "Press Enter to migrate and continue...")
		bufio.NewReader(in).ReadString('\n')
		fmt.Fprintln(out)
	}
	if err := os.MkdirAll(filepath.Dir(newDir), 0o755); err != nil {
		return fmt.Errorf("creating config parent dir: %w", err)
	}
	if err := os.Rename(oldDir, newDir); err != nil {
		// A concurrent spekk process (e.g. agent shims launching in parallel)
		// may have completed the migration between our check and the rename.
		if !fsutil.DirExists(oldDir) && fsutil.DirExists(newDir) {
			return nil
		}
		// Cross-filesystem rename fails; fall back to copy + delete, staged
		// through a temp dir so a crash mid-copy never leaves a partial
		// config dir at the final path — the old dir stays intact and the
		// migration simply retries on the next run.
		tmpDir := newDir + ".migrating"
		os.RemoveAll(tmpDir)
		if err2 := copyDir(oldDir, tmpDir); err2 != nil {
			return fmt.Errorf("migrating config dir: %w", err2)
		}
		if err2 := os.Rename(tmpDir, newDir); err2 != nil {
			return fmt.Errorf("migrating config dir: %w", err2)
		}
		if err2 := os.RemoveAll(oldDir); err2 != nil {
			return fmt.Errorf("removing old config dir after migration: %w", err2)
		}
	}
	fmt.Fprintf(out, "Migrated %s → %s\n\n", oldDir, newDir)
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
