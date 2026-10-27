package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/spekk-ai/spekk-cli/internal/agentconn"
)

// The X25519 test vector of RFC 7748 section 6.1 (the key pair of Alice), so
// the expected public key does not come from the code under test.
const (
	vectorPrivateB64 = "dwdtCnMYpX08FsFyUbJmRd9ML4frwJkqsXf7pR25LCo="
	vectorPublicHex  = "8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a"
	vectorPublicB64  = "hSDwCYkwp1R0i33ctD73Wg2/Og0mOBr066SpjqqbTmo="
	// vectorFingerprint is pinned, so spekk-app can use the same vector.
	vectorFingerprint = "300c9c9603b92a4b"
)

func writeVectorKey(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spekk-agent", "secret-key")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(vectorPrivateB64+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readRecord(t *testing.T, path string) (agentconn.Record, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r agentconn.Record
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	return r, string(data)
}

// The fingerprint is what an admin compares with the Secrets section before
// accepting the key, so it must be the value that spekk-app computes.
func TestFingerprintMatchesTheTestVector(t *testing.T) {
	pub, err := hex.DecodeString(vectorPublicHex)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(pub)
	if want := hex.EncodeToString(sum[:])[:16]; want != vectorFingerprint {
		t.Fatalf("the pinned fingerprint %s is not SHA-256 of the vector public key (%s)", vectorFingerprint, want)
	}

	record := filepath.Join(t.TempDir(), "agent-connection.json")
	NewAgentClient(Config{SecretKeyPath: writeVectorKey(t), ConnectionRecord: record})
	if got, _ := readRecord(t, record); got.SecretKeyFingerprint != vectorFingerprint {
		t.Errorf("record fingerprint = %q, want %q", got.SecretKeyFingerprint, vectorFingerprint)
	}
}

// A restart must keep the key: a new key needs a new accept by an admin, and
// every secret sealed to the old key is lost.
func TestKeyIsPrivateAndSurvivesARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spekk-agent", "secret-key")
	record := filepath.Join(t.TempDir(), "agent-connection.json")

	first := NewAgentClient(Config{SecretKeyPath: path, ConnectionRecord: record})
	if first.secretKey == nil {
		t.Fatal("the first start made no key")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("key file mode = %04o, want 0600", perm)
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dir.Mode().Perm(); perm != 0o700 {
		t.Errorf("key directory mode = %04o, want 0700", perm)
	}
	firstRecord, _ := readRecord(t, record)

	second := NewAgentClient(Config{SecretKeyPath: path, ConnectionRecord: record})
	if second.secretKey == nil || !bytes.Equal(first.secretKey.PublicKey().Bytes(), second.secretKey.PublicKey().Bytes()) {
		t.Fatal("a restart did not load the same key")
	}
	secondRecord, raw := readRecord(t, record)
	if secondRecord.SecretKeyFingerprint == "" || secondRecord.SecretKeyFingerprint != firstRecord.SecretKeyFingerprint {
		t.Errorf("fingerprint changed over a restart: %q, then %q", firstRecord.SecretKeyFingerprint, secondRecord.SecretKeyFingerprint)
	}
	assertNoPrivateKey(t, "the connection record", raw, path)
}

// A key file that is not safe is refused and kept. A silent new key would
// make an admin accept a key that nobody expected.
func TestUnsafeKeyFileIsRefusedAndKept(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, path string)
		reason string
	}{
		{"world-readable", func(t *testing.T, path string) {
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatal(err)
			}
		}, "mode 0644"},
		{"bad contents", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("not a key\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "does not hold a key"},
		{"symbolic link", func(t *testing.T, path string) {
			target := path + ".real"
			if err := os.Rename(path, target); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}, "symbolic link"},
		// A blocking open of a FIFO waits for a writer with no time limit.
		{"fifo", func(t *testing.T, path string) {
			if err := os.Rename(path, path+".real"); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal(err)
			}
		}, "not a regular file"},
		// No key file yet: the agent must not make a key that other users
		// can replace.
		{"world-writable directory", func(t *testing.T, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(filepath.Dir(path), 0o777); err != nil {
				t.Fatal(err)
			}
		}, "mode 0777"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeVectorKey(t)
			tt.setup(t, path)
			before := keyFileState(t, path)
			record := filepath.Join(t.TempDir(), "agent-connection.json")
			logs := captureLog(t)

			loaded := make(chan *AgentClient, 1)
			go func() { loaded <- NewAgentClient(Config{SecretKeyPath: path, ConnectionRecord: record}) }()
			var client *AgentClient
			select {
			case client = <-loaded:
			case <-time.After(5 * time.Second):
				t.Fatal("the agent did not finish the key load in 5 seconds")
			}

			if client.secretKey != nil {
				t.Fatal("the agent used a key file that it must refuse")
			}
			if after := keyFileState(t, path); before != after {
				t.Errorf("the agent changed the refused key file: %q, then %q", before, after)
			}
			got, raw := readRecord(t, record)
			if got.SecretKeyFingerprint != "" || !strings.Contains(got.SecretKeyError, tt.reason) {
				t.Errorf("record = %+v, want no fingerprint and an error with %q", got, tt.reason)
			}
			if !strings.Contains(logs.String(), path) || !strings.Contains(logs.String(), tt.reason) {
				t.Errorf("the log does not name the file and the reason:\n%s", logs)
			}
			assertNoPrivateKey(t, "the log", logs.String(), path)
			assertNoPrivateKey(t, "the connection record", raw, path)
		})
	}
}

// The real client against a control host that does not know secrets_hello:
// the key goes out after each welcome, the answer is logged one time, and
// the connection keeps working.
func TestSecretsHelloAndAnOldControlHost(t *testing.T) {
	keyPath := writeVectorKey(t)
	var hellos []string
	served := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		ctx := r.Context()
		// Three welcomes, each answered as an old control host answers. The
		// third hello proves that the client still reads and writes after
		// the second answer.
		for i := 0; i < 3; i++ {
			if err := wsjson.Write(ctx, conn, map[string]any{"type": "welcome", "protocol": ProtocolVersion, "agent_id": "agent-7"}); err != nil {
				return
			}
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			hellos = append(hellos, string(data))
			if i < 2 {
				if err := wsjson.Write(ctx, conn, map[string]any{"type": "error", "message": "Unknown message type: secrets_hello"}); err != nil {
					return
				}
			}
		}
		close(served)
		<-ctx.Done()
	}))
	t.Cleanup(srv.Close)

	logs := captureLog(t)
	host := "localhost:" + strings.TrimPrefix(srv.URL, "http://127.0.0.1:")
	client := NewAgentClient(Config{Token: "t", Host: host, Workspace: t.TempDir(), SecretKeyPath: keyPath})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { client.Run(ctx); close(done) }()
	awaitRecord(t, "the client did not send secrets_hello after each welcome", func() bool {
		select {
		case <-served:
			return true
		default:
			return false
		}
	})
	cancel()
	<-done

	for _, hello := range hellos {
		var frame struct {
			Type      string          `json:"type"`
			PublicKey string          `json:"public_key"`
			Held      json.RawMessage `json:"held"`
		}
		if err := json.Unmarshal([]byte(hello), &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type != "secrets_hello" || frame.PublicKey != vectorPublicB64 || string(frame.Held) != "{}" {
			t.Errorf("frame = %s, want secrets_hello with public_key %s and held {}", hello, vectorPublicB64)
		}
		assertNoPrivateKey(t, "secrets_hello", hello, keyPath)
	}
	out := logs.String()
	if n := strings.Count(out, "does not support secrets"); n != 1 {
		t.Errorf("the old control host was logged %d times, want 1:\n%s", n, out)
	}
	if !strings.Contains(out, vectorFingerprint) {
		t.Errorf("the log has no fingerprint line:\n%s", out)
	}
	if strings.Contains(out, "Connection lost") {
		t.Errorf("the answer of the old control host ended the connection:\n%s", out)
	}
	assertNoPrivateKey(t, "the log", out, keyPath)

	if got := client.secretAgentID(); got != "agent-7" {
		t.Errorf("seal agent id = %q, want the agent_id of welcome", got)
	}
	client.cfg.AgentID = "from-env"
	if got := client.secretAgentID(); got != "from-env" {
		t.Errorf("seal agent id = %q, want SPEKK_AGENT_ID before the agent_id of welcome", got)
	}
}

// keyFileState is the contents of a regular file, or else its type. It does
// not open a FIFO, because that open waits for a writer.
func keyFileState(t *testing.T, path string) string {
	t.Helper()
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "no file"
	}
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		return info.Mode().Type().String()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// assertNoPrivateKey fails when text holds the private key of the key file,
// in base64 or in hex.
func assertNoPrivateKey(t *testing.T, what, text, keyPath string) {
	t.Helper()
	if info, err := os.Lstat(keyPath); err == nil && !info.Mode().IsRegular() {
		// A refused symbolic link or FIFO: the key is in the .real file.
		keyPath += ".real"
	}
	data, err := os.ReadFile(keyPath)
	if errors.Is(err, fs.ErrNotExist) {
		return // no key file, so there is no key to leak
	}
	if err != nil {
		t.Fatal(err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return // not a key, so there is nothing to leak
	}
	for _, form := range []string{strings.TrimSpace(string(data)), hex.EncodeToString(key)} {
		if strings.Contains(text, form) {
			t.Fatalf("%s contains the private key", what)
		}
	}
}
