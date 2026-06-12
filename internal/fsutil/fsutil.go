// Package fsutil provides small filesystem helpers shared across packages.
package fsutil

import "os"

// DirExists reports whether path exists and is a directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
