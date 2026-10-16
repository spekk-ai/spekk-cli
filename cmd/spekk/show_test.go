package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/show"
)

// TestExecShow_Help_ExitZero checks that --help prints the show usage to
// stdout, exits 0, and calls neither the explorer build nor the watch server,
// also when --watch comes with it.
func TestExecShow_Help_ExitZero(t *testing.T) {
	fail := func(name string) func(string, show.Options) error {
		return func(string, show.Options) error {
			t.Errorf("--help called %s", name)
			return nil
		}
	}
	runners := showRunners{run: fail("show.Run"), watch: fail("show.RunWatch")}

	for _, args := range [][]string{{"--help"}, {"-h"}, {"--watch", "--help"}} {
		var stdout bytes.Buffer
		if code := execShow(args, &stdout, io.Discard, runners); code != 0 {
			t.Errorf("%v: exit code %d, want 0", args, code)
		}
		if stdout.String() != showUsage {
			t.Errorf("%v: stdout is not the show usage: %q", args, stdout.String())
		}
	}
}
