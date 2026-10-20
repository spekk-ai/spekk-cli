package sandbox

import (
	"errors"
	"strings"
	"testing"
)

func TestEnsureCodexCLIRepairsFailedHealthCheck(t *testing.T) {
	for _, tc := range []struct {
		name      string
		repairErr error
		wantErr   string
	}{
		{name: "repair succeeds"},
		{name: "repair fails", repairErr: errors.New("download failed"), wantErr: "installing Codex CLI: download failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := sshExec
			t.Cleanup(func() { sshExec = original })
			calls := 0
			var commands []string
			sshExec = func(args []string, _ string) ([]byte, error) {
				calls++
				commands = append(commands, args[len(args)-1])
				if calls == 1 {
					return nil, errors.New("host check failed")
				}
				return []byte("codex-cli " + codexVersion + "\n"), tc.repairErr
			}

			err := ensureCodexCLI("127.0.0.1", "", "test", "root")
			if calls != 2 {
				t.Errorf("SSH calls = %d, want health check and repair", calls)
			} else if commands[0] != codexHealthCommand || commands[1] != codexInstallScript {
				t.Error("SSH did not run the health check before repair")
			}
			if tc.wantErr == "" && err != nil {
				t.Fatalf("repair: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("repair error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}
