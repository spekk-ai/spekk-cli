//go:build !linux && !darwin

package sandbox

import (
	"fmt"
	"os"
	"runtime"
)

func openCodexAuthFile(string) (*os.File, error) {
	return nil, fmt.Errorf("secure Codex auth file loading is unavailable on %s", runtime.GOOS)
}
