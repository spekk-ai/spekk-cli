//go:build !windows

package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// fakeClaude writes a claude stand-in into dir that records its arguments,
// separated by NUL because the message has newlines, to the file in
// $FAKE_CLAUDE_RECORD.
func fakeClaude(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "claude")
	script := "#!/bin/sh\nprintf '%s\\0' \"$@\" > \"$FAKE_CLAUDE_RECORD\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func readRecordedArgs(t *testing.T, record string) []string {
	t.Helper()
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatalf("fake claude did not run: %v", err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
}

// claude reads --mcp-config as a list, so the flag must arrive as one
// "--mcp-config=<file>" token, or claude reads the message as a second file.
func TestRunObserver_PassesMCPConfigToClaude(t *testing.T) {
	install := setupObserverSkill(t, "coverage-gap", "# Coverage Gap Skill")
	cli.DefaultEmbeddedFS = testEmbeddedFS("observer")
	t.Cleanup(func() { cli.DefaultEmbeddedFS = nil })
	bin := t.TempDir()
	claude := fakeClaude(t, bin)
	record := filepath.Join(bin, "args")
	t.Setenv("FAKE_CLAUDE_RECORD", record)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cases := []struct {
		name string
		args []string
		want []string // arguments before the message
		msg  string   // text in the message, the last argument
	}{
		{"headless scan", []string{"--headless", "--claude-path", claude, "--mcp-config", "mcp.json"},
			[]string{"-p", "--dangerously-skip-permissions", "--mcp-config=mcp.json"}, "# observer base prompt"},
		{"headless skill", []string{"coverage-gap", "--headless", "--claude-path", claude, "--mcp-config", "mcp.json"},
			[]string{"-p", "--dangerously-skip-permissions", "--mcp-config=mcp.json"}, "# Coverage Gap Skill"},
		{"interactive scan", []string{"--mcp-config", "mcp.json"},
			[]string{"--dangerously-skip-permissions", "--mcp-config=mcp.json"}, "# observer base prompt"},
		{"headless scan without the flag", []string{"--headless", "--claude-path", claude},
			[]string{"-p", "--dangerously-skip-permissions"}, "# observer base prompt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A new project for each case: a launch keeps its lock open for
			// the life of the process. The project is not the isolated home,
			// or .spekk reads as an old global config directory and moves.
			t.Chdir(t.TempDir())
			if err := os.Mkdir(".spekk", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("mcp.json", []byte(`{"mcpServers":{}}`), 0o644); err != nil {
				t.Fatal(err)
			}
			os.Remove(record)
			RunObserver(tc.args, install)

			got := readRecordedArgs(t, record)
			if len(got) != len(tc.want)+1 {
				t.Fatalf("claude got %d arguments, want %d: %q", len(got), len(tc.want)+1, got)
			}
			for i, w := range tc.want {
				if got[i] != w {
					t.Errorf("argument %d: got %q, want %q", i, got[i], w)
				}
			}
			if !strings.Contains(got[len(got)-1], tc.msg) {
				t.Errorf("the last argument is not the activation message: %q", got[len(got)-1])
			}
		})
	}
}

// A --mcp-config that names no file must stop the run before claude starts,
// so a scheduled run never goes on without the tools it asked for.
// RunObserver exits the process, so each case runs in a child test binary.
func TestRunObserver_RefusesBadMCPConfig(t *testing.T) {
	if os.Getenv("SPEKK_TEST_RUN_OBSERVER_MCP") == "1" {
		// os.Exit skips t.TempDir cleanup, so the skill install leaks one
		// temp directory for each case. The case must exit before a launch.
		install := setupObserverSkill(t, "coverage-gap", "# Coverage Gap Skill")
		RunObserver(strings.Split(os.Getenv("SPEKK_TEST_RUN_OBSERVER_ARGS"), " "), install)
		return
	}

	bin := t.TempDir()
	claude := fakeClaude(t, bin)
	record := filepath.Join(bin, "args")

	cases := []struct {
		name    string
		args    string
		message string
	}{
		{"missing file", "--headless --claude-path " + claude + " --mcp-config " + filepath.Join(bin, "absent.json"),
			"--mcp-config file not found: " + filepath.Join(bin, "absent.json")},
		{"directory", "--headless --claude-path " + claude + " --mcp-config " + bin,
			"--mcp-config file not found: " + bin},
		{"missing value", "--headless --claude-path " + claude + " --mcp-config",
			"--mcp-config requires a file"},
		{"missing value in a skill run", "coverage-gap --headless --claude-path " + claude + " --mcp-config",
			"--mcp-config requires a file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run", "^TestRunObserver_RefusesBadMCPConfig$")
			cmd.Env = append(os.Environ(),
				"SPEKK_TEST_RUN_OBSERVER_MCP=1",
				"SPEKK_TEST_RUN_OBSERVER_ARGS="+tc.args,
				"FAKE_CLAUDE_RECORD="+record,
				"PATH="+bin)
			out, err := cmd.CombinedOutput()

			if _, ok := err.(*exec.ExitError); !ok {
				t.Fatalf("run did not exit non-zero: %v; output:\n%s", err, out)
			}
			if !strings.Contains(string(out), tc.message) {
				t.Errorf("output does not contain %q:\n%s", tc.message, out)
			}
			if _, err := os.Stat(record); err == nil {
				t.Errorf("claude started although --mcp-config was bad")
			}
		})
	}
}
