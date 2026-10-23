package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/agentconn"
)

// maxFailureErrorLen caps the error text in the record. A handshake error is
// one line; the cap only keeps a long one from filling the status output.
const maxFailureErrorLen = 200

// connectionRecorder keeps the record of this agent's handshakes on the
// machine, so that `spekk sandbox status` can tell a refused agent from a
// connected one (specs/agent-handshake-visible/). Only Run's goroutine uses
// it, one dial at a time, so it has no lock.
type connectionRecorder struct {
	path   string // empty: keep no record
	record agentconn.Record
	warned bool
}

// newConnectionRecorder starts from the record on the disk, so a restart
// keeps the last success. A missing record starts empty, and so does one
// that cannot be read, with a log line that says why.
func newConnectionRecorder(path string) connectionRecorder {
	r := connectionRecorder{path: path}
	if path == "" {
		return r
	}
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &r.record)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		r.record = agentconn.Record{}
		log.Printf("WARNING: could not read the connection record %s: %v. It starts again empty.", path, err)
	}
	return r
}

func (r *connectionRecorder) connected(at time.Time) {
	r.record.LastConnectedAt = at.UTC().Format(agentconn.TimeLayout)
	r.write()
}

// failed records a dial that did not complete the handshake. resp is the
// HTTP response of the handshake, which is nil when the host did not answer.
func (r *connectionRecorder) failed(at time.Time, resp *http.Response, err error) {
	r.record.LastFailureAt = at.UTC().Format(agentconn.TimeLayout)
	r.record.LastFailureStatus = 0
	if resp != nil {
		r.record.LastFailureStatus = resp.StatusCode
	}
	r.record.LastFailureError = err.Error()
	if len(r.record.LastFailureError) > maxFailureErrorLen {
		r.record.LastFailureError = r.record.LastFailureError[:maxFailureErrorLen]
	}
	r.write()
}

// write replaces the record in one step. A failure does not stop the agent:
// the record is for a person who reads the status, not for the agent. It is
// logged once for each process, because the dial loop would repeat it every
// minute.
func (r *connectionRecorder) write() {
	if r.path == "" {
		return
	}
	if err := writeRecord(r.path, r.record); err != nil && !r.warned {
		r.warned = true
		log.Printf("WARNING: could not write the connection record %s: %v. spekk sandbox status cannot show the handshake state.", r.path, err)
	}
}

func writeRecord(path string, record agentconn.Record) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".agent-connection-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(append(data, '\n'))
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	// CreateTemp makes the file 0600, and the person who runs status reads
	// it over SSH as a user other than agent. It holds no secret.
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
