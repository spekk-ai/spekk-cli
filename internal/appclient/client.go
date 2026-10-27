package appclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Host string `json:"host"`
	Key  string `json:"key"`
}

type Client struct {
	host string
	key  string
	http *http.Client
}

type APIError struct {
	Method string
	Route  string
	Status int
	Reason string
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Route, e.Status, e.Reason)
	if e.Status == http.StatusUnauthorized {
		message += "; create a new personal API key in Settings and run spekk app auth set"
	}
	return message
}

func ValidateHost(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("app host must be an origin without a path, query, fragment, or user info")
	}
	if u.Scheme == "https" {
		return u.String(), nil
	}
	if u.Scheme == "http" {
		if ip := net.ParseIP(u.Hostname()); ip != nil && ip.IsLoopback() {
			return u.String(), nil
		}
		if strings.EqualFold(u.Hostname(), "localhost") {
			return u.String(), nil
		}
	}
	return "", errors.New("app host must use HTTPS, except for loopback development")
}

// appDir holds the key and the operation records. It is a private directory
// of its own, because other spekk commands make the parent spekk directory
// with mode 0755.
func appDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "spekk", "app"), nil
}

func ConfigPath() (string, error) {
	dir, err := appDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

func SaveConfig(host, key string) error {
	valid, err := ValidateHost(host)
	if err != nil {
		return err
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 4096 {
		return errors.New("personal API key is empty or too long")
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return writePrivate(path, Config{Host: valid, Key: key})
}

func writePrivate(path string, value any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.New("app credential directory must be private")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".app-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func LoadConfig() (Config, error) {
	var config Config
	path, err := ConfigPath()
	if err != nil {
		return config, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return config, fmt.Errorf("app key is not set; run spekk app auth set: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return config, errors.New("app credential file must be a private regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, errors.New("app credential file is invalid")
	}
	if _, err := ValidateHost(config.Host); err != nil {
		return Config{}, err
	}
	if config.Key == "" {
		return Config{}, errors.New("app credential file has no key")
	}
	return config, nil
}

func New(config Config) (*Client, error) {
	host, err := ValidateHost(config.Host)
	if err != nil {
		return nil, err
	}
	if config.Key == "" {
		return nil, errors.New("app credential file has no key")
	}
	transport := &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 20 * time.Second}
	return &Client{host: host, key: config.Key, http: &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("redirect refused")
		},
	}}, nil
}

func (c *Client) Do(ctx context.Context, method, route string, body any) (map[string]any, error) {
	if !strings.HasPrefix(route, "/api/") || strings.HasPrefix(route, "//") {
		return nil, errors.New("invalid app API route")
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.host+route, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %s", method, route, strings.ReplaceAll(err.Error(), c.key, "[redacted]"))
	}
	defer resp.Body.Close()
	limit := int64(1 << 20)
	if resp.StatusCode >= 300 {
		limit = 4096
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%s %s: read response: %w", method, route, err)
	}
	if int64(len(data)) > limit {
		if resp.StatusCode >= 300 {
			return nil, &APIError{Method: method, Route: route, Status: resp.StatusCode, Reason: "response body is too large"}
		}
		return nil, fmt.Errorf("%s %s: response is too large", method, route)
	}
	var result map[string]any
	if len(data) > 0 {
		if err := json.Unmarshal(data, &result); err != nil && resp.StatusCode < 300 {
			return nil, fmt.Errorf("%s %s: invalid JSON response", method, route)
		}
	}
	if resp.StatusCode >= 300 {
		reason, _ := result["detail"].(string)
		if reason == "" {
			reason = strings.TrimSpace(string(data))
		}
		if reason == "" {
			reason = http.StatusText(resp.StatusCode)
		}
		return nil, &APIError{Method: method, Route: route, Status: resp.StatusCode, Reason: strings.ReplaceAll(reason, c.key, "[redacted]")}
	}
	return result, nil
}
