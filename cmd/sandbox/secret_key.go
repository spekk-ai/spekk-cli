package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// The recipient key of the secrets that the control host seals to this
// sandbox (specs/sandbox-secret-recipient-key/). The private key stays in its
// file: it is never logged, printed, or sent. Only the public key and its
// fingerprint leave this process.

// secretKeyPath is where the agent keeps its private key, in the config
// directory of the agent user: /home/agent/.config/spekk-agent/secret-key on
// a sandbox. The image hygiene script and the destroy of a machine that
// stays remove this directory.
func secretKeyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "spekk-agent", "secret-key"), nil
}

// keyFingerprint is the value that an organization admin compares with the
// Secrets section of spekk-app before it accepts the key: the first 16 hex
// characters of SHA-256 of the 32 public key bytes.
func keyFingerprint(publicKey []byte) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:])[:16]
}

// loadRecipientKey reads the private key at path, or makes one when there is
// no file. It never replaces a file that is there: a file that is not safe or
// not a key is an error, and a person decides what to do with it. A silent
// new key would make an admin accept a key that nobody expected.
func loadRecipientKey(path string) (*ecdh.PrivateKey, error) {
	key, err := readRecipientKey(path)
	if errors.Is(err, fs.ErrNotExist) {
		return createRecipientKey(path)
	}
	return key, err
}

// readRecipientKey reads the key file. No error text holds the contents.
func readRecipientKey(path string) (*ecdh.PrivateKey, error) {
	// O_NOFOLLOW: a symbolic link is not the file of this agent.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%s is a symbolic link, not a regular file", path)
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return nil, fmt.Errorf("%s has mode %04o, so other users can read or change it; it must be 0600", path, perm)
	}
	// A key is 45 bytes. The limit only keeps a wrong file out of memory.
	data, err := io.ReadAll(io.LimitReader(f, 1024))
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(raw) != 32 {
		return nil, fmt.Errorf("%s does not hold a key (standard base64 of 32 bytes)", path)
	}
	return ecdh.X25519().NewPrivateKey(raw)
}

// createRecipientKey makes a new key pair and writes it to path with mode
// 0600, in a directory with mode 0700. The file is complete before it gets
// its name, and the link does not replace a file that another process wrote
// first: the key of that file is used.
func createRecipientKey(path string) (*ecdh.PrivateKey, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	// CreateTemp makes the file 0600.
	tmp, err := os.CreateTemp(dir, ".secret-key-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	_, werr := tmp.WriteString(base64.StdEncoding.EncodeToString(key.Bytes()) + "\n")
	serr := tmp.Sync()
	cerr := tmp.Close()
	if err := errors.Join(werr, serr, cerr); err != nil {
		return nil, err
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return readRecipientKey(path)
		}
		return nil, err
	}
	return key, nil
}

// secretsHelloFrame announces the public key after each welcome. held is
// empty: this client applies no secret yet.
func secretsHelloFrame(publicKey []byte) map[string]any {
	return map[string]any{
		"type":       MessageTypeSecretsHello,
		"public_key": base64.StdEncoding.EncodeToString(publicKey),
		"held":       map[string]int{},
	}
}
