package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func downloadAttachments(cfg Config, attachments []Attachment) []string {
	if len(attachments) == 0 {
		return nil
	}

	scheme := "https"
	if contains(cfg.Host, "localhost") {
		scheme = "http"
	}

	subdir := filepath.Join(cfg.Workspace, ".attachments", fmt.Sprintf("%d", time.Now().UnixMilli()))
	if err := os.MkdirAll(subdir, 0755); err != nil {
		log.Printf("Failed to create attachments dir: %v", err)
		return nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var paths []string

	for i, a := range attachments {
		if a.ID == "" {
			log.Printf("Attachment missing ID, skipping: %s", a.Filename)
			continue
		}

		// TODO: Extract these to constants?
		url := fmt.Sprintf("%s://%s/api/agents/attachments/%s/download/", scheme, cfg.Host, a.ID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
		req.Header.Set("User-Agent", "SpekkAgent/1.0")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to download attachment %s: %v", a.Filename, err)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Failed to read attachment %s: %v", a.Filename, err)
			continue
		}

		filename := fmt.Sprintf("%d-%s", i+1, a.Filename)
		path := filepath.Join(subdir, filename)
		if err := os.WriteFile(path, data, 0644); err != nil {
			log.Printf("Failed to save attachment %s: %v", a.Filename, err)
			continue
		}

		log.Printf("Saved attachment: %s (%d bytes)", path, len(data))
		paths = append(paths, path)
	}

	return paths
}
