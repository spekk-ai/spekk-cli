package verify

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/anchors"
	"github.com/spekk-ai/spekk-cli/internal/index"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// DisplayValue is the recorded truth at HEAD and its strongest passing level.
type DisplayValue struct {
	Truth string
	Level string
}

const DirtyNote = "⚠️ Results are from a run with uncommitted changes"

// DisplayValues reads the latest run at HEAD. Prose without a marker uses the
// same anchor rule as verify, also when no run exists.
func DisplayValues(specsDir string, assertions []parser.Assertion) (map[string]DisplayValue, bool, error) {
	root := filepath.Dir(specsDir)
	head, err := anchors.Head(root)
	if err != nil {
		return nil, false, err
	}
	values := make(map[string]DisplayValue, len(assertions))
	for _, a := range assertions {
		values[a.ID] = DisplayValue{Truth: "unchecked"}
	}
	if head == "" {
		return values, false, nil
	}
	dbPath := index.DBPath(root)
	if rebuilt, err := index.EnsureFresh(specsDir, dbPath); err != nil {
		return nil, false, err
	} else if rebuilt {
		_ = index.EnsureGitignored(root)
	}
	quotedHead := "'" + escapeSQL(head) + "'"
	query := `SELECT t.assertion_id, t.truth, t.level FROM verify_truth t JOIN verify_runs r ON r.id=t.run_id WHERE r.id=(SELECT MAX(id) FROM verify_runs WHERE "commit"=` + quotedHead + `)`
	rows, err := index.RunQuery(dbPath, query)
	if err != nil {
		return nil, false, err
	}
	for _, row := range rows.Rows {
		if _, ok := values[row[0]]; ok {
			values[row[0]] = DisplayValue{Truth: row[1], Level: row[2]}
		}
	}
	markers, err := index.RunQuery(dbPath, `SELECT DISTINCT assertion_id FROM evidence`)
	if err != nil {
		return nil, false, err
	}
	marked := map[string]bool{}
	for _, row := range markers.Rows {
		marked[row[0]] = true
	}
	db, err := sql.Open("sqlite", index.ReadDSN(dbPath))
	if err != nil {
		return nil, false, err
	}
	defer db.Close()
	for _, a := range assertions {
		if marked[a.ID] {
			continue
		}
		changed, err := anchorChanged(db, root, property{id: a.ID, file: a.File}, head)
		if err != nil {
			return nil, false, fmt.Errorf("truth for %s: %w", a.ID, err)
		}
		if changed {
			values[a.ID] = DisplayValue{Truth: "unconfirmed"}
		}
	}
	dirtyRows, err := index.RunQuery(dbPath, `SELECT dirty FROM verify_runs WHERE "commit"=`+quotedHead+` ORDER BY id DESC LIMIT 1`)
	if err != nil {
		return nil, false, err
	}
	return values, len(dirtyRows.Rows) > 0 && dirtyRows.Rows[0][0] == "1", nil
}

func escapeSQL(value string) string {
	// Git commit IDs contain only hexadecimal digits. Keep this helper so the
	// query stays safe if a future HEAD representation changes.
	return strings.ReplaceAll(value, "'", "''")
}

// FillDisplayValues sets transient display fields after parsing.
func FillDisplayValues(specsDir string, assertions []parser.Assertion) (bool, error) {
	values, dirty, err := DisplayValues(specsDir, assertions)
	if err != nil {
		return false, err
	}
	for i := range assertions {
		v := values[assertions[i].ID]
		assertions[i].Truth, assertions[i].Level = v.Truth, v.Level
	}
	return dirty, nil
}
