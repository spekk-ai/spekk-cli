package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// The sign-in terminal (specs/sandbox-sign-in-terminal/). The owner of a
// sandbox with no model credential signs in to their own Claude account with
// the login of the unmodified Claude Code binary, and the control host relays
// the bytes between this terminal and a browser.
//
// No frame names a program: the terminal runs claude auth login and nothing
// else. No byte of the terminal goes to a log or a file, because the login
// shows a sign-in URL and the person types a code.

const (
	MessageTypeTerminalOpen   = "terminal_open"
	MessageTypeTerminalInput  = "terminal_input"
	MessageTypeTerminalResize = "terminal_resize"
	MessageTypeTerminalClose  = "terminal_close"
	MessageTypeTerminalOutput = "terminal_output"
	MessageTypeTerminalExit   = "terminal_exit"
)

// signInTimeout ends a sign-in that nobody finishes. A variable, so that a
// test can make it short.
var signInTimeout = 10 * time.Minute

// signInArgs are the arguments of the one program that the terminal runs.
var signInArgs = []string{"auth", "login"}

// signInTerminal is one running login.
type signInTerminal struct {
	id      string
	agentID string
	userID  string
	ptmx    *os.File
	cancel  context.CancelFunc
}

// terminals holds the one open terminal. A second open ends the first, so a
// terminal that a closed browser left behind never blocks a new sign-in.
type terminals struct {
	mu   sync.Mutex
	open *signInTerminal
}

// replace makes t the open terminal and ends the terminal before it, if any.
func (ts *terminals) replace(t *signInTerminal) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.open != nil {
		ts.open.cancel()
	}
	ts.open = t
}

// get returns the open terminal when its id is id, else nil. Input for a
// terminal that has ended goes nowhere.
func (ts *terminals) get(id string) *signInTerminal {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.open == nil || ts.open.id != id {
		return nil
	}
	return ts.open
}

// remove forgets t, unless a newer terminal took its place.
func (ts *terminals) remove(t *signInTerminal) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.open == t {
		ts.open = nil
	}
}

func terminalOutputFrame(id string, data []byte) map[string]any {
	return map[string]any{
		"type":        MessageTypeTerminalOutput,
		"terminal_id": id,
		"data":        base64.StdEncoding.EncodeToString(data),
	}
}

func terminalExitFrame(id string, code int) map[string]any {
	return map[string]any{"type": MessageTypeTerminalExit, "terminal_id": id, "exit_code": code}
}

func heartbeatFrame() map[string]any {
	return map[string]any{"type": MessageTypeHeartbeat, "claude_signed_in": claudeSignedIn()}
}

func (c *AgentClient) handleTerminal(msg Message) {
	switch msg.Type {
	case MessageTypeTerminalOpen:
		c.openTerminal(msg)
	case MessageTypeTerminalInput:
		if t := c.terms.get(msg.TerminalID); t != nil {
			data, err := base64.StdEncoding.DecodeString(msg.Data)
			if err == nil {
				_, _ = t.ptmx.Write(data)
			}
		}
	case MessageTypeTerminalResize:
		if t := c.terms.get(msg.TerminalID); t != nil {
			_ = pty.Setsize(t.ptmx, terminalSize(msg))
		}
	case MessageTypeTerminalClose:
		if t := c.terms.get(msg.TerminalID); t != nil {
			t.cancel()
		}
	}
}

func terminalSize(msg Message) *pty.Winsize {
	size := &pty.Winsize{Cols: 80, Rows: 24}
	if msg.Cols > 0 && msg.Cols <= 1000 {
		size.Cols = uint16(msg.Cols)
	}
	if msg.Rows > 0 && msg.Rows <= 1000 {
		size.Rows = uint16(msg.Rows)
	}
	return size
}

// openTerminal starts claude auth login in a pseudo-terminal, as the user of
// this process, and relays it until it ends.
func (c *AgentClient) openTerminal(msg Message) {
	id := msg.TerminalID
	path, err := exec.LookPath("claude")
	if err != nil {
		_ = c.conns.send(c.procCtx, terminalOutputFrame(id, []byte("Claude Code is not on the PATH of this sandbox.\r\n")))
		_ = c.conns.send(c.procCtx, terminalExitFrame(id, 127))
		return
	}

	ctx, cancel := context.WithTimeout(c.procCtx, signInTimeout)
	cmd := exec.CommandContext(ctx, path, signInArgs...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	// pty.Start makes the login the leader of a new session, so its pid is
	// also its process group. An end stops the whole group: the login and
	// each process that it started.
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	ptmx, err := pty.StartWithSize(cmd, terminalSize(msg))
	if err != nil {
		cancel()
		log.Printf("Sign-in terminal did not start (agent %s, user %s, terminal %s): %v", msg.AgentID, msg.UserID, id, err)
		_ = c.conns.send(c.procCtx, terminalExitFrame(id, -1))
		return
	}

	t := &signInTerminal{id: id, agentID: msg.AgentID, userID: msg.UserID, ptmx: ptmx, cancel: cancel}
	c.terms.replace(t)
	opened := time.Now()
	log.Printf("Sign-in terminal opened (agent %s, user %s, terminal %s)", t.agentID, t.userID, id)

	go func() {
		relayed := make(chan struct{})
		go func() {
			defer close(relayed)
			buf := make([]byte, 4096)
			for {
				n, err := ptmx.Read(buf)
				if n > 0 {
					// Dropped when no connection is live: a terminal
					// shows the present, and the person can open it again.
					_ = c.conns.send(c.procCtx, terminalOutputFrame(id, buf[:n]))
				}
				if err != nil {
					return
				}
			}
		}()

		code := exitCode(cmd.Wait())
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		// The last bytes of the login are in the terminal until the read
		// gets them. A process that holds the terminal open must not hold
		// this goroutine.
		select {
		case <-relayed:
		case <-time.After(2 * time.Second):
		}
		ptmx.Close()
		cancel()
		c.terms.remove(t)
		log.Printf("Sign-in terminal closed (agent %s, user %s, terminal %s, %s, exit code %d)",
			t.agentID, t.userID, id, time.Since(opened).Round(time.Second), code)

		// The heartbeat goes first, so the control host has the new sign-in
		// state when the browser hears of the exit.
		_ = c.conns.send(c.procCtx, heartbeatFrame())
		_ = c.conns.sendFinal(c.procCtx, terminalExitFrame(id, code))
	}()
}

// exitCode is the exit status of the login, or -1 when a signal stopped it
// or it did not run.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
