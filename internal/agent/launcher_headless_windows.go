//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
)

// LaunchHeadless spawns the harness in headless/print (non-interactive) mode
// with the given message. It does not inherit stdin and does not forward
// signals — suitable for background invocations where no TTY is present.
//
// binaryPath is an explicit path to the harness binary; if empty, the profile's
// binary name is used (relies on PATH).
//
// On Windows the overlap guard (flock) is not available; the lockFile parameter
// is accepted for API compatibility but ignored.
func LaunchHeadless(profile Profile, binaryPath, lockFile, message string) error {
	if binaryPath == "" {
		binaryPath = profile.Binary
	}

	cmd := exec.Command(binaryPath, profile.HeadlessArgs(message)...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = headlessChildEnv()

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			l1, l2 := profile.notFoundLines()
			fmt.Fprintln(os.Stderr, "Error: "+l1)
			fmt.Fprintln(os.Stderr, l2)
			os.Exit(1)
		}
		return fmt.Errorf("Error launching %s headless: %w", profile.DisplayName, err)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
