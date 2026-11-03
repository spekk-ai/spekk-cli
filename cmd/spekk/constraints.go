package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/anchors"
	"github.com/spekk-ai/spekk-cli/internal/formatter"
	"github.com/spekk-ai/spekk-cli/internal/parser"
	"github.com/spekk-ai/spekk-cli/internal/verify"
)

type constraintRow struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Truth  string `json:"truth"`
	Level  string `json:"level"`
	Lines  []int  `json:"lines"`
	Title  string `json:"title"`

	priority int
}

func runConstraints(args []string) {
	if code := execConstraints(args, os.Stdout, os.Stderr, findSpecsDir()); code != 0 {
		os.Exit(code)
	}
}

func execConstraints(args []string, stdout, stderr io.Writer, specsDir string) int {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(stdout, "Usage: spekk constraints <file> [--json]\nList properties that anchor on a project file.")
		return 0
	}
	var path string
	jsonMode := false
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonMode = true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintf(stderr, "Error: unknown option %q\n", arg)
			return 1
		case path == "":
			path = arg
		default:
			fmt.Fprintln(stderr, "Error: give one file")
			return 1
		}
	}
	if path == "" {
		fmt.Fprintln(stderr, "Error: give a file\nUsage: spekk constraints <file> [--json]")
		return 1
	}
	root, err := filepath.Abs(filepath.Dir(specsDir))
	if err != nil {
		return constraintError(stderr, err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return constraintError(stderr, err)
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return constraintError(stderr, fmt.Errorf("file is outside project root: %s", path))
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return constraintError(stderr, err)
	}
	for candidate := absPath; ; candidate = filepath.Dir(candidate) {
		resolved, resolveErr := filepath.EvalSymlinks(candidate)
		if resolveErr == nil {
			resolvedRel, relErr := filepath.Rel(resolvedRoot, resolved)
			if relErr != nil || resolvedRel == ".." || strings.HasPrefix(resolvedRel, ".."+string(filepath.Separator)) {
				return constraintError(stderr, fmt.Errorf("file is outside project root: %s", path))
			}
			break
		}
		if !os.IsNotExist(resolveErr) {
			return constraintError(stderr, resolveErr)
		}
		if candidate == root || candidate == filepath.Dir(candidate) {
			break
		}
	}
	rel = filepath.ToSlash(rel)
	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return constraintError(stderr, err)
	}
	// spekk: constraints-reads-current-file
	scan, err := anchors.ScanFile(root, rel)
	if err != nil {
		return constraintError(stderr, err)
	}
	byID := make(map[string]*constraintRow)
	// spekk: constraints-lists-anchored-properties
	for _, a := range result.Assertions {
		for _, anchor := range a.Anchors {
			if filepath.ToSlash(filepath.Clean(anchor)) == rel {
				byID[a.ID] = &constraintRow{ID: a.ID, Status: a.Status, Lines: []int{}, Title: a.Title, priority: a.Priority}
				break
			}
		}
	}
	known := make(map[string]parser.Assertion, len(result.Assertions))
	for _, a := range result.Assertions {
		known[a.ID] = a
	}
	for _, anchor := range scan.Anchors {
		a, ok := known[anchor.ID]
		if !ok {
			fmt.Fprintf(stderr, "Warning: %s:%d: unknown anchor ID %q\n", rel, anchor.Line, anchor.ID)
			continue
		}
		row := byID[a.ID]
		if row == nil {
			row = &constraintRow{ID: a.ID, Status: a.Status, Lines: []int{}, Title: a.Title, priority: a.Priority}
			byID[a.ID] = row
		}
		row.Lines = append(row.Lines, anchor.Line)
	}
	rows := make([]constraintRow, 0, len(byID))
	for _, row := range byID {
		slices.Sort(row.Lines)
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].priority != rows[j].priority {
			return rows[i].priority < rows[j].priority
		}
		return rows[i].ID < rows[j].ID
	})
	if len(rows) == 0 {
		if jsonMode {
			fmt.Fprintln(stdout, "[]")
		} else {
			fmt.Fprintf(stderr, "no properties anchor on %s\n", rel)
		}
		return 0
	}
	values, _, err := verify.DisplayValues(specsDir, result.Assertions)
	if err != nil {
		return constraintError(stderr, err)
	}
	for i := range rows {
		rows[i].Truth = values[rows[i].ID].Truth
		rows[i].Level = values[rows[i].ID].Level
	}
	if jsonMode {
		if err := json.NewEncoder(stdout).Encode(rows); err != nil {
			return constraintError(stderr, err)
		}
		return 0
	}
	matrix := [][]string{{"ID", "STATUS", "TRUTH", "LINES", "TITLE"}}
	for _, row := range rows {
		lines := make([]string, len(row.Lines))
		for i, line := range row.Lines {
			lines[i] = strconv.Itoa(line)
		}
		matrix = append(matrix, []string{row.ID, row.Status, formatter.TruthCell(row.Truth, row.Level), strings.Join(lines, ", "), row.Title})
	}
	fmt.Fprintln(stdout, formatter.RenderTable(matrix))
	return 0
}

func constraintError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "Error: %s\n", err)
	return 1
}
