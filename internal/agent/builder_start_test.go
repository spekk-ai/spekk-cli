package agent

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestBuilderInterruptDuringChildStart(t *testing.T) {
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	interrupted := filepath.Join(dir, "interrupted")
	child := "#!/bin/sh\ntrap 'printf interrupted > \"$INTERRUPTED_FILE\"; exit 130' INT\nprintf ready > \"$READY_FILE\"\ni=0\nwhile [ \"$i\" -lt 40 ]; do sleep 0.05; i=$((i + 1)); done\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(child), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("READY_FILE", ready)
	t.Setenv("INTERRUPTED_FILE", interrupted)

	started := make(chan struct{})
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	holder := &processHolder{afterStart: func() {
		close(started)
		<-release
	}}
	finished := make(chan error, 1)
	go func() {
		_, err := launchClaude(nil, holder)
		finished <- err
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("child did not start")
	}
	deadline := time.After(3 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("child did not become ready")
		case <-time.After(10 * time.Millisecond):
		}
	}
	handled := make(chan bool, 1)
	go func() {
		active := holder.isSet()
		if active {
			holder.signal(syscall.SIGINT)
		}
		handled <- active
	}()
	var active bool
	select {
	case active = <-handled:
		close(release)
	case <-time.After(50 * time.Millisecond):
		close(release)
		select {
		case active = <-handled:
		case <-time.After(3 * time.Second):
			t.Fatal("signal handler did not finish")
		}
	}
	if !active {
		t.Fatal("holder treated a started child as idle")
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("builder did not wait for the interrupted child")
	}
	if _, err := os.Stat(interrupted); err != nil {
		t.Fatalf("child did not receive SIGINT: %v", err)
	}
}
