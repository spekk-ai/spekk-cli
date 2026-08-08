package crossbranch

import (
	"encoding/json"
	"strconv"
)

// fileStateJSON is the machine-readable shape of one Classify contribution.
// Meta is deliberately excluded: it carries full file content for the HTML
// explorer, and the machine surface promises only the classification row
// (path, branch, state, degraded, status drift).
type fileStateJSON struct {
	Path      string `json:"path"`
	Branch    string `json:"branch"`
	State     string `json:"state"`
	Degraded  bool   `json:"degraded"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// FormatJSON renders Classify results as a JSON array of objects. An empty
// or nil slice renders as [], never null, so jq pipelines see an array.
//
// Every row carries every key, empty statuses included. Omitting them would
// give the same surface two shapes -- TSV and CSV always emit the columns --
// so a consumer switching format would have to handle a missing key in one
// and an empty cell in the other.
func FormatJSON(states []FileState) (string, error) {
	rows := make([]fileStateJSON, 0, len(states))
	for _, s := range states {
		rows = append(rows, fileStateJSON{
			Path:      s.Path,
			Branch:    s.Branch,
			State:     string(s.State),
			Degraded:  s.Degraded,
			OldStatus: s.OldStatus,
			NewStatus: s.NewStatus,
		})
	}
	out, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// OutputColumns is the column set for the table/TSV/CSV renderings of
// Classify results, matching the JSON keys.
var OutputColumns = []string{"path", "branch", "state", "degraded", "old_status", "new_status"}

// OutputRows converts Classify results into string cells in OutputColumns
// order, for the shared formatter renderers.
func OutputRows(states []FileState) [][]string {
	rows := make([][]string, 0, len(states))
	for _, s := range states {
		rows = append(rows, []string{
			s.Path,
			s.Branch,
			string(s.State),
			strconv.FormatBool(s.Degraded),
			s.OldStatus,
			s.NewStatus,
		})
	}
	return rows
}
