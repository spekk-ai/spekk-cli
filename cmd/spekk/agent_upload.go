package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type uploadFile struct {
	path string
	name string
	size int64
}

func agentBase(getenv func(string) string) (string, string, error) {
	host, token := getenv("SPEKK_HOST"), getenv("SPEKK_AGENT_TOKEN")
	if host == "" || token == "" {
		return "", "", errors.New("SPEKK_HOST and SPEKK_AGENT_TOKEN must be set in a sandbox session")
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return "", "", errors.New("SPEKK_HOST must be an HTTPS host or a loopback HTTP host")
	}
	return strings.TrimRight(u.String(), "/"), token, nil
}

func postAgentFiles(getenv func(string) string, endpoint string, fields map[string]string, files []uploadFile) (int, map[string]any, error) {
	base, token, err := agentBase(getenv)
	if err != nil {
		return 0, nil, err
	}
	reader, writer := io.Pipe()
	form := multipart.NewWriter(writer)
	go func() {
		for key, value := range fields {
			if err := form.WriteField(key, value); err != nil {
				writer.CloseWithError(err)
				return
			}
		}
		for _, file := range files {
			input, err := os.Open(file.path)
			if err != nil {
				writer.CloseWithError(err)
				return
			}
			part, err := form.CreateFormFile("file", file.name)
			if err == nil {
				_, err = io.Copy(part, input)
			}
			input.Close()
			if err != nil {
				writer.CloseWithError(err)
				return
			}
		}
		if err := form.Close(); err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()
	req, err := http.NewRequest(http.MethodPost, base+endpoint, reader)
	if err != nil {
		reader.Close()
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", form.FormDataContentType())
	client := &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		reader.Close()
		return 0, nil, fmt.Errorf("agent upload failed: %w", err)
	}
	defer response.Body.Close()
	var result map[string]any
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return response.StatusCode, nil, errors.New("the app returned an invalid upload response")
	}
	return response.StatusCode, result, nil
}

func regularUpload(path string, limit int64) (uploadFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return uploadFile{}, err
	}
	if !info.Mode().IsRegular() {
		return uploadFile{}, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > limit {
		return uploadFile{}, fmt.Errorf("%s is over the %d MB limit", path, limit/(1024*1024))
	}
	return uploadFile{path: path, name: filepath.Base(path), size: info.Size()}, nil
}

func responseText(result map[string]any) string {
	if detail, ok := result["detail"].(string); ok && detail != "" {
		return detail
	}
	return "the app refused the request"
}
