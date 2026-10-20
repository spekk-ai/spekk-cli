//go:build linux || darwin

package sandbox

import (
	"fmt"
	"os"
	"syscall"
)

func openCodexAuthFile(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("checking Codex auth file: %w", err)
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("checking Codex auth file: %w", err)
	}
	owner, ok := info.Sys().(*syscall.Stat_t)
	if !ok || !info.Mode().IsRegular() || owner.Uid != uint32(os.Getuid()) || info.Mode().Perm()&0o077 != 0 {
		f.Close()
		return nil, fmt.Errorf("Codex auth file must be a private regular file owned by the current user")
	}
	return f, nil
}
