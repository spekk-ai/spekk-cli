package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	maxAttachmentBytes          = 20 << 20
	maxTotalAttachmentBytes     = 40 << 20
	maxAttachmentCount          = 10
	maxAttachmentNameBytes      = 200
	maxAttachmentExtensionBytes = 16
	attachmentHTTPTimeout       = 30 * time.Second
)

var attachmentIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func attachmentBaseURL(host string) (*url.URL, error) {
	base, err := url.Parse("https://" + host)
	if err != nil || base.Host != host || base.Path != "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("invalid attachment host")
	}
	ip := net.ParseIP(base.Hostname())
	if base.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()) {
		base.Scheme = "http"
	}
	return base, nil
}

func safeAttachmentName(name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("invalid attachment filename")
	}
	safe := strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._- ", char) {
			return char
		}
		return '_'
	}, name)
	if len(safe) > maxAttachmentNameBytes {
		extension := filepath.Ext(safe)
		if len(extension) > maxAttachmentExtensionBytes {
			extension = ""
		}
		safe = safe[:maxAttachmentNameBytes-len(extension)] + extension
	}
	return safe, nil
}

func downloadAttachments(ctx context.Context, cfg Config, attachments []Attachment) (paths []string, err error) {
	if len(attachments) == 0 {
		return nil, nil
	}
	if len(attachments) > maxAttachmentCount {
		return nil, fmt.Errorf("too many attachments")
	}
	base, err := attachmentBaseURL(cfg.Host)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(attachments))
	for index, attachment := range attachments {
		if !attachmentIDPattern.MatchString(attachment.ID) {
			return nil, fmt.Errorf("attachment %d has an invalid ID", index+1)
		}
		names[index], err = safeAttachmentName(attachment.Filename)
		if err != nil {
			return nil, fmt.Errorf("attachment %d: %w", index+1, err)
		}
	}
	root := filepath.Join(cfg.Workspace, ".attachments")
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("create attachments directory: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("attachments directory is not a regular directory")
	}
	subdir, err := os.MkdirTemp(root, "turn-")
	if err != nil {
		return nil, fmt.Errorf("create turn attachments directory: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			if cleanupErr := os.RemoveAll(subdir); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("remove partial attachments: %w", cleanupErr))
			}
		}
	}()
	client := &http.Client{
		Timeout:       attachmentHTTPTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	paths = make([]string, 0, len(attachments))
	var total int64
	for index, attachment := range attachments {
		target := *base
		target.Path = fmt.Sprintf("/api/agents/attachments/%s/download/", attachment.ID)
		path := filepath.Join(subdir, fmt.Sprintf("%d-%s", index+1, names[index]))
		budget := min(int64(maxAttachmentBytes), int64(maxTotalAttachmentBytes)-total)
		size, err := downloadAttachment(ctx, client, target.String(), cfg.Token, path, budget)
		if err != nil {
			return nil, fmt.Errorf("download attachment %d: %w", index+1, err)
		}
		total += size
		paths = append(paths, path)
	}
	complete = true
	return paths, nil
}

func downloadAttachment(ctx context.Context, client *http.Client, target, token, path string, budget int64) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "SpekkAgent/1.0")
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("attachment server returned HTTP %d", response.StatusCode)
	}
	if response.ContentLength > budget {
		return 0, fmt.Errorf("attachment exceeds the download limit")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return 0, err
	}
	size, copyErr := io.Copy(file, io.LimitReader(response.Body, budget+1))
	closeErr := file.Close()
	if copyErr != nil {
		return 0, copyErr
	}
	if closeErr != nil {
		return 0, closeErr
	}
	if size == 0 || size > budget {
		return 0, fmt.Errorf("attachment is empty or exceeds the download limit")
	}
	return size, nil
}
