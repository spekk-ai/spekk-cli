package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/show"
)

// TestExecShow_Help_ExitZero checks that --help prints the show usage to
// stdout, exits 0, and calls none of the explorer build, the watch server, and
// the JSON output, also when --watch or --json comes with it.
func TestExecShow_Help_ExitZero(t *testing.T) {
	fail := func(name string) func(string, show.Options) error {
		return func(string, show.Options) error {
			t.Errorf("--help called %s", name)
			return nil
		}
	}
	runners := showRunners{run: fail("show.Run"), watch: fail("show.RunWatch"), json: func(io.Writer, string, show.Options) error {
		t.Error("--help called show.WriteJSON")
		return nil
	}}

	for _, args := range [][]string{{"--help"}, {"-h"}, {"--watch", "--help"}, {"--json", "--help"}} {
		var stdout bytes.Buffer
		if code := execShow(args, &stdout, io.Discard, runners); code != 0 {
			t.Errorf("%v: exit code %d, want 0", args, code)
		}
		if stdout.String() != showUsage {
			t.Errorf("%v: stdout is not the show usage: %q", args, stdout.String())
		}
	}
}
