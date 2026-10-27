package appclient

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type Operation struct {
	Kind        string    `json:"kind"`
	RequestID   string    `json:"request_id"`
	Host        string    `json:"host"`
	LaunchID    string    `json:"launch_id,omitempty"`
	AgentID     string    `json:"agent_id,omitempty"`
	MessageID   string    `json:"message_id,omitempty"`
	PayloadHash string    `json:"payload_hash,omitempty"`
	Created     time.Time `json:"created"`
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NewRequestID returns a random UUID. The app chat routes accept only a UUID.
func NewRequestID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16]), nil
}

// TaskRequestID returns id in the lowercase UUID form that the app stores.
func TaskRequestID(id string) (string, error) {
	id = strings.ToLower(id)
	if !uuidPattern.MatchString(id) {
		return "", errors.New("a send request ID must be a UUID")
	}
	return id, nil
}

func operationsDir() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "operations"), nil
}

func SaveOperation(op Operation) error {
	if !requestIDPattern.MatchString(op.RequestID) {
		return errors.New("request ID must use 1 to 128 letters, digits, dots, underscores, or hyphens")
	}
	dir, err := operationsDir()
	if err != nil {
		return err
	}
	if op.Created.IsZero() {
		op.Created = time.Now().UTC()
	}
	path := filepath.Join(dir, op.RequestID+".json")
	if old, err := LoadOperation(op.RequestID); err == nil {
		if old.Kind != op.Kind || old.Host != op.Host || old.PayloadHash != op.PayloadHash {
			return fmt.Errorf("request ID %s belongs to different operation fields", op.RequestID)
		}
		if old.LaunchID != "" && op.LaunchID != "" && old.LaunchID != op.LaunchID || old.AgentID != "" && op.AgentID != "" && old.AgentID != op.AgentID {
			return fmt.Errorf("request ID %s belongs to a different sandbox", op.RequestID)
		}
		if op.LaunchID == "" {
			op.LaunchID = old.LaunchID
		}
		if op.AgentID == "" {
			op.AgentID = old.AgentID
		}
		if op.MessageID == "" {
			op.MessageID = old.MessageID
		}
		op.Created = old.Created
	} else if !os.IsNotExist(err) {
		return err
	}
	return writePrivate(path, op)
}

func LoadOperation(requestID string) (Operation, error) {
	var op Operation
	dir, err := operationsDir()
	if err != nil {
		return op, err
	}
	if !requestIDPattern.MatchString(requestID) {
		return op, errors.New("invalid request ID")
	}
	path := filepath.Join(dir, requestID+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return op, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return op, errors.New("operation must be a private regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return op, err
	}
	if err := json.Unmarshal(data, &op); err != nil {
		return op, errors.New("operation file is invalid")
	}
	return op, nil
}

func ListOperations() ([]Operation, error) {
	dir, err := operationsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Operation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return nil, fmt.Errorf("operation %s is not a private regular file", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var op Operation
		if err := json.Unmarshal(data, &op); err != nil {
			return nil, fmt.Errorf("operation %s is invalid", entry.Name())
		}
		result = append(result, op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Created.Before(result[j].Created) })
	return result, nil
}
