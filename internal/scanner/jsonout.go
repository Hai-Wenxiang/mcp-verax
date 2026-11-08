package scanner

import (
	"encoding/json"
	"io"
)

// RenderJSON writes the full result as machine-readable JSON (CI tooling,
// dashboards, diffing scans over time).
func RenderJSON(w io.Writer, res *Result) error {
	out := struct {
		Target   string         `json:"target"`
		Root     string         `json:"root,omitempty"`
		Files    int            `json:"files"`
		Score    int            `json:"score"`
		Counts   map[string]int `json:"counts"`
		Findings []Finding      `json:"findings"`
	}{
		Target:   res.Target,
		Root:     res.Root,
		Files:    res.Files,
		Score:    res.Score(),
		Counts:   map[string]int{},
		Findings: res.Findings,
	}
	if out.Findings == nil {
		out.Findings = []Finding{}
	}
	for sev, n := range res.CountsPerSeverity() {
		out.Counts[string(sev)] = n
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
