package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const artifactUsage = `spekk artifact upload <file> [--title <text>] [--kind <markdown|image|video>]

Upload a file with the sandbox session's agent token. The artifact starts with organization access.
`

func runArtifact(args []string) {
	if err := execArtifact(args, os.Stdout, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func execArtifact(args []string, stdout io.Writer, getenv func(string) string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "help" || (len(args) > 1 && args[0] == "upload" && args[1] == "--help") {
		_, _ = io.WriteString(stdout, artifactUsage)
		return nil
	}
	if args[0] != "upload" || len(args) < 2 {
		return fmt.Errorf("use %s", strings.TrimSpace(strings.Split(artifactUsage, "\n")[0]))
	}
	path, title, kind := args[1], "", ""
	for i := 2; i < len(args); i++ {
		if (args[i] != "--title" && args[i] != "--kind") || i+1 == len(args) || args[i+1] == "" {
			return fmt.Errorf("expected --title or --kind with a value")
		}
		i++
		if args[i-1] == "--title" {
			title = args[i]
		} else {
			kind = args[i]
		}
	}
	if kind == "" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown":
			kind = "markdown"
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			kind = "image"
		case ".mp4":
			kind = "video"
		default:
			return fmt.Errorf("cannot infer the artifact kind; use --kind")
		}
	}
	limits := map[string]int64{"markdown": 1 << 20, "image": 10 << 20, "video": 90 << 20}
	limit, ok := limits[kind]
	if !ok {
		return fmt.Errorf("kind must be markdown, image, or video")
	}
	file, err := regularUpload(path, limit)
	if err != nil {
		return err
	}
	if title == "" {
		title = file.name
	}
	status, result, err := postAgentFiles(getenv, "/api/agents/artifacts/", map[string]string{"kind": kind, "title": title}, []uploadFile{file})
	if err != nil {
		return err
	}
	if status == 404 {
		return fmt.Errorf("artifacts are off on this app")
	}
	if status != 201 {
		return fmt.Errorf("artifact upload failed: %s", responseText(result))
	}
	link, _ := result["app_url"].(string)
	if link == "" {
		return fmt.Errorf("the app returned no artifact link")
	}
	_, _ = fmt.Fprintln(stdout, link)
	return nil
}
