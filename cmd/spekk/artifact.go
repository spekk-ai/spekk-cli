package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const artifactUsage = `spekk artifact upload <file> [--title <text>] [--kind <markdown|text|image|video>]

Upload a file with the sandbox session's agent token. The artifact starts with organization access.
The first line is the direct link to share. The second line is the app page, where a person changes the access.
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
		switch ext := strings.ToLower(filepath.Ext(path)); ext {
		case ".md", ".markdown":
			kind = "markdown"
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			kind = "image"
		case ".mp4":
			kind = "video"
		case ".txt", ".log", ".csv", ".tsv", ".json", ".jsonl", ".yaml", ".yml", ".toml", ".ini", ".sql", ".py", ".go", ".js", ".ts", ".tsx", ".jsx", ".sh", ".rb", ".java", ".kt", ".rs", ".c", ".h", ".cpp", ".css", ".graphql", ".proto", ".diff", ".patch":
			kind = "text"
		default:
			if strings.HasSuffix(strings.ToLower(path), ".env.example") {
				kind = "text"
			} else if ext == ".html" || ext == ".htm" || ext == ".svg" || ext == ".xhtml" {
				return fmt.Errorf("cannot infer the artifact kind; active content is not a text artifact")
			} else {
				valid, err := isUTF8File(path)
				if err != nil {
					return err
				}
				if !valid {
					return fmt.Errorf("cannot infer the artifact kind; use --kind")
				}
				kind = "text"
			}
		}
	}
	limits := map[string]int64{"markdown": 1 << 20, "text": 1 << 20, "image": 10 << 20, "video": 90 << 20}
	limit, ok := limits[kind]
	if !ok {
		return fmt.Errorf("kind must be markdown, text, image, or video")
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
		if kind == "text" && status == 400 && (strings.Contains(responseText(result), "Choose the kind") || strings.Contains(responseText(result), "Unknown kind")) {
			return fmt.Errorf("this app does not support text artifacts yet: %s", responseText(result))
		}
		return fmt.Errorf("artifact upload failed: %s", responseText(result))
	}
	view, _ := result["view_url"].(string)
	app, _ := result["app_url"].(string)
	if view == "" {
		// An older app returns no view_url: print app_url alone, as before.
		view, app = app, ""
	}
	if view == "" {
		return fmt.Errorf("the app returned no artifact link")
	}
	_, _ = fmt.Fprintln(stdout, view)
	if app != "" {
		_, _ = fmt.Fprintf(stdout, "App page: %s\n", app)
	}
	return nil
}

func isUTF8File(path string) (bool, error) {
	if _, err := regularUpload(path, 1<<20); err != nil {
		return false, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return utf8.Valid(content) && !bytes.Contains(content, []byte{0}), nil
}
