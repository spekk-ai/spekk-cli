package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/show"
)

// TestExecShow_Help_ExitZero checks that --help prints the show usage, exits 0,
// and calls neither the explorer build nor the watch server, also when other
// flags come with it.
func TestExecShow_Help_ExitZero(t *testing.T) {
	fail := func(name string) func(string, show.Options) error {
		return func(string, show.Options) error {
			t.Errorf("--help called %s", name)
			return nil
		}
	}
	runners := showRunners{run: fail("show.Run"), watch: fail("show.RunWatch")}

	for _, args := range [][]string{{"--help"}, {"-h"}, {"--watch", "--help"}, {"--cross-branch", "-h"}} {
		var stdout bytes.Buffer
		if code := execShow(args, &stdout, io.Discard, runners); code != 0 {
			t.Errorf("%v: exit code %d, want 0", args, code)
		}
		for _, want := range []string{"spekk show", "--watch", "--cross-branch", "--branch-filter", "--help"} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("%v: usage does not contain %q: %q", args, want, stdout.String())
			}
		}
	}
}
