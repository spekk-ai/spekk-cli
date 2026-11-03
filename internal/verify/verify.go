// Package verify records JUnit evidence and property truth for a project commit.
package verify

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/anchors"
	"github.com/spekk-ai/spekk-cli/internal/index"
	_ "modernc.org/sqlite"
)

const usage = `spekk verify - Run project checks and record property truth at HEAD

USAGE:
  spekk verify [--help]

Configure one or more [[verify.run]] entries in spekk.toml at the project root.
Each entry needs a command and a junit path or glob.
`

type marker struct{ id, level, language, file, test string }
type testResult struct{ class, name, outcome string }
type property struct{ id, file, status, kind string }
type resultRow struct{ id, file, test, level, outcome string }
type truthRow struct {
	truth, level string
	failed       []string
}

var errFailed = errors.New("verification failed")

// Run runs the configured commands, then records their results in the index.
func Run(specsDir string, args []string, stdout, stderr io.Writer) error {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		_, _ = io.WriteString(stdout, usage)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("use spekk verify --help for usage")
	}
	root := filepath.Dir(specsDir)
	config, problems := ReadConfig(filepath.Join(root, "spekk.toml"))
	if len(problems) != 0 {
		for _, problem := range problems {
			fmt.Fprintln(stderr, "spekk.toml:", problem)
		}
		return ErrInvalidConfig
	}
	configs := config.Runs
	if len(configs) == 0 {
		return fmt.Errorf("configure a [[verify.run]] command and junit path in spekk.toml")
	}
	head, err := anchors.Head(root)
	if err != nil {
		return err
	}
	dirty := 0
	if head != "" {
		out, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=normal").Output()
		if err != nil {
			return fmt.Errorf("cannot check work tree: %w", err)
		}
		if len(out) != 0 {
			dirty = 1
		}
	}
	if dirty != 0 {
		if _, err := index.BuildIndex(specsDir, index.DBPath(root), false); err != nil {
			return err
		}
	} else if _, err := index.EnsureFresh(specsDir, index.DBPath(root)); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", index.DBPath(root))
	if err != nil {
		return err
	}
	defer db.Close()
	markers, properties, err := readIndex(db)
	if err != nil {
		return err
	}
	module := goModule(root)
	started := time.Now()
	freshSince, err := fsNow(root)
	if err != nil {
		return err
	}
	var results []resultRow
	seen := map[int]bool{}
	failed := false
	for number, cfg := range configs {
		cmd := exec.Command("sh", "-c", cfg.command)
		cmd.Dir = root
		cmd.Stdout, cmd.Stderr = stderr, stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(stderr, "command %d failed: %v\n", number+1, err)
			failed = true
		}
		paths, err := filepath.Glob(filepath.Join(root, cfg.junit))
		if err != nil {
			return fmt.Errorf("invalid junit glob %q: %w", cfg.junit, err)
		}
		if len(paths) == 0 {
			fmt.Fprintf(stderr, "%s: not written by this run\n", cfg.junit)
			failed = true
		}
		for _, path := range paths {
			if err := insideRoot(root, path); err != nil {
				return err
			}
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() || info.ModTime().Before(freshSince) {
				fmt.Fprintf(stderr, "%s: not written by this run\n", rel(root, path))
				failed = true
				continue
			}
			cases, err := readJUnit(path)
			if err != nil {
				fmt.Fprintf(stderr, "%s: %v\n", rel(root, path), err)
				failed = true
				continue
			}
			for _, c := range cases {
				for i, m := range markers {
					if !matches(m, c, module) {
						continue
					}
					seen[i] = true
					results = append(results, resultRow{m.id, m.file, m.test, m.level, c.outcome})
				}
			}
		}
	}
	for i, m := range markers {
		if !seen[i] {
			fmt.Fprintf(stderr, "no result: %s:%s\n", m.file, m.test)
		}
	}
	truth := make(map[string]truthRow, len(properties))
	for _, p := range properties {
		hasMarker := false
		for _, m := range markers {
			if m.id == p.id {
				hasMarker = true
				break
			}
		}
		v := truthRow{truth: "unchecked"}
		if !hasMarker && head != "" {
			changed, err := anchorChanged(db, root, p, head)
			if err != nil {
				return err
			}
			if changed {
				v.truth = "unconfirmed"
			}
		}
		truth[p.id] = v
	}
	for _, r := range results {
		v, ok := truth[r.id]
		if !ok {
			continue
		}
		switch r.outcome {
		case "failed", "error":
			v.truth = "failing"
			v.failed = append(v.failed, r.test)
		case "passed":
			if v.truth != "failing" {
				v.truth = "verified"
			}
			if strength(r.level) > strength(v.level) {
				v.level = r.level
			}
		}
		truth[r.id] = v
	}
	// A failure has priority even when a later result passes.
	for _, r := range results {
		if r.outcome == "failed" || r.outcome == "error" {
			v := truth[r.id]
			v.truth = "failing"
			truth[r.id] = v
		}
	}
	if err := record(db, head, dirty, started, time.Now(), results, truth); err != nil {
		return err
	}
	strictFailed := false
	for _, p := range properties {
		if p.status != "done" {
			continue
		}
		v := truth[p.id]
		if v.truth == "unchecked" || v.truth == "unconfirmed" {
			message := fmt.Sprintf("strict: %s is %s", p.id, v.truth)
			if config.Unchecked == "error" {
				fmt.Fprintln(stderr, message)
				strictFailed = true
			} else {
				fmt.Fprintln(stderr, "warning: "+message)
			}
		}
		if minimum := config.MinLevel[p.kind]; minimum != "" && strength(v.level) < strength(minimum) {
			level := v.level
			if level == "" {
				level = "no level"
			}
			fmt.Fprintf(stderr, "strict: %s is %s, below %s for %s\n", p.id, level, minimum, p.kind)
			strictFailed = true
		}
	}
	counts := map[string]int{}
	ids := make([]string, 0, len(truth))
	for id := range truth {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		v := truth[id]
		counts[v.truth]++
		if v.truth == "failing" {
			sort.Strings(v.failed)
			fmt.Fprintf(stdout, "%s: failing: %s\n", id, strings.Join(v.failed, ", "))
			failed = true
		}
	}
	fmt.Fprintf(stdout, "verify: %d verified, %d failing, %d unchecked, %d unconfirmed\n", counts["verified"], counts["failing"], counts["unchecked"], counts["unconfirmed"])
	if failed {
		return errFailed
	}
	if strictFailed {
		return ErrStrictFailed
	}
	return nil
}

func fsNow(root string) (time.Time, error) {
	dir := filepath.Join(root, ".spekk")
	f, err := os.CreateTemp(dir, "verify-clock-")
	if err != nil {
		return time.Time{}, fmt.Errorf("create clock file in %s: %w", dir, err)
	}
	path := f.Name()
	info, statErr := f.Stat()
	closeErr := f.Close()
	removeErr := os.Remove(path)
	if statErr != nil {
		return time.Time{}, fmt.Errorf("stat %s: %w", path, statErr)
	}
	if closeErr != nil {
		return time.Time{}, fmt.Errorf("close %s: %w", path, closeErr)
	}
	if removeErr != nil {
		return time.Time{}, fmt.Errorf("remove %s: %w", path, removeErr)
	}
	return info.ModTime(), nil
}

func readIndex(db *sql.DB) ([]marker, []property, error) {
	rows, err := db.Query(`SELECT assertion_id, level, language, file, test FROM evidence`)
	if err != nil {
		return nil, nil, err
	}
	var markers []marker
	for rows.Next() {
		var m marker
		if err := rows.Scan(&m.id, &m.level, &m.language, &m.file, &m.test); err != nil {
			rows.Close()
			return nil, nil, err
		}
		markers = append(markers, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = db.Query(`SELECT id, file, status, kind FROM assertions ORDER BY id`)
	if err != nil {
		return nil, nil, err
	}
	var properties []property
	for rows.Next() {
		var p property
		if err := rows.Scan(&p.id, &p.file, &p.status, &p.kind); err != nil {
			rows.Close()
			return nil, nil, err
		}
		properties = append(properties, p)
	}
	err = rows.Err()
	rows.Close()
	return markers, properties, err
}

func goModule(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func matches(m marker, c testResult, module string) bool {
	name := strings.SplitN(c.name, "/", 2)[0]
	name = strings.SplitN(name, "[", 2)[0]
	dir := filepath.ToSlash(filepath.Dir(m.file))
	if m.language == "go" {
		pkg := module
		if dir != "." {
			pkg += "/" + dir
		}
		return c.class == pkg && name == m.test
	}
	if m.language == "python" {
		path := strings.TrimSuffix(filepath.ToSlash(m.file), ".py")
		path = strings.ReplaceAll(path, "/", ".")
		parts := strings.Split(m.test, ".")
		if len(parts) == 2 {
			return c.class == path+"."+parts[0] && name == parts[1]
		}
		return c.class == path && name == m.test
	}
	return false
}

func readJUnit(path string) ([]testResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 32<<20+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("JUnit file is too large")
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var out []testResult
	var current *testResult
	depth := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.Directive:
			return nil, fmt.Errorf("JUnit directives are not allowed")
		case xml.StartElement:
			if t.Name.Local == "testcase" {
				v := testResult{outcome: "passed"}
				for _, a := range t.Attr {
					if a.Name.Local == "classname" {
						v.class = a.Value
					}
					if a.Name.Local == "name" {
						v.name = a.Value
					}
				}
				current = &v
				depth = 1
			} else if current != nil {
				depth++
				if depth == 2 {
					switch t.Name.Local {
					case "failure":
						current.outcome = "failed"
					case "error":
						current.outcome = "error"
					case "skipped":
						current.outcome = "skipped"
					}
				}
			}
		case xml.EndElement:
			if current != nil {
				depth--
				if depth == 0 {
					out = append(out, *current)
					current = nil
				}
			}
		}
	}
	return out, nil
}

func insideRoot(root, path string) error {
	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	absPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("JUnit file is outside the project: %s", path)
	}
	return nil
}
func rel(root, path string) string {
	p, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(p)
}
func strength(level string) int {
	switch level {
	case "example":
		return 1
	case "property":
		return 2
	case "contract":
		return 3
	case "bounded":
		return 4
	case "proof":
		return 5
	}
	return 0
}

func anchorChanged(db *sql.DB, root string, p property, head string) (bool, error) {
	rows, err := db.Query(`SELECT DISTINCT path FROM anchors WHERE assertion_id = ?`, p.id)
	if err != nil {
		return false, err
	}
	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return false, err
		}
		paths = append(paths, path)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(paths) == 0 {
		return false, err
	}
	last, err := exec.Command("git", "-C", root, "log", "-1", "--format=%H", "--", p.file).Output()
	if err != nil {
		return false, err
	}
	commit := strings.TrimSpace(string(last))
	if commit == "" || commit == head {
		return false, nil
	}
	args := append([]string{"-C", root, "diff", "--name-only", commit, head, "--"}, paths...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) != 0, nil
}

func record(db *sql.DB, head string, dirty int, started, finished time.Time, results []resultRow, truth map[string]truthRow) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO verify_runs ("commit", dirty, started, finished) VALUES (?, ?, ?, ?)`, head, dirty, started.UTC().Format(time.RFC3339Nano), finished.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for _, r := range results {
		if _, err := tx.Exec(`INSERT INTO verify_results (run_id, assertion_id, file, test, level, outcome) VALUES (?, ?, ?, ?, ?, ?)`, id, r.id, r.file, r.test, r.level, r.outcome); err != nil {
			return err
		}
	}
	for key, v := range truth {
		if _, err := tx.Exec(`INSERT INTO verify_truth (run_id, assertion_id, truth, level) VALUES (?, ?, ?, ?)`, id, key, v.truth, v.level); err != nil {
			return err
		}
	}
	return tx.Commit()
}
