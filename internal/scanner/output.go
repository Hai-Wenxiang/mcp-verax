package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Hai-Wenxiang/mcp-verax/internal/version"
)

// sarifLevel maps severities onto SARIF levels.
func sarifLevel(s Severity) string {
	switch s {
	case SevCritical, SevHigh:
		return "error"
	case SevMedium:
		return "warning"
	}
	return "note"
}

type sarifDoc struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool struct {
		Driver struct {
			Name    string      `json:"name"`
			Version string      `json:"version"`
			Rules   []sarifRule `json:"rules"`
		} `json:"driver"`
	} `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifRule struct {
	ID               string `json:"id"`
	ShortDescription struct {
		Text string `json:"text"`
	} `json:"shortDescription"`
}

type sarifResult struct {
	RuleID  string `json:"ruleId"`
	Level   string `json:"level"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				Uri string `json:"uri"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
}

// RenderSARIF writes the scan result as SARIF 2.1.0 (GitHub Security tab compatible).
func RenderSARIF(w io.Writer, res *Result) error {
	doc := sarifDoc{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
	}
	doc.Runs = append(doc.Runs, sarifRun{})

	ruleSeen := map[string]bool{}
	for _, f := range res.Findings {
		if ruleSeen[f.RuleID] {
			continue
		}
		ruleSeen[f.RuleID] = true
		r := sarifRule{ID: f.RuleID}
		r.ShortDescription.Text = f.Title
		doc.Runs[0].Tool.Driver.Rules = append(doc.Runs[0].Tool.Driver.Rules, r)
	}
	doc.Runs[0].Tool.Driver.Name = "verax"
	doc.Runs[0].Tool.Driver.Version = version.Version

	for _, f := range res.Findings {
		var out sarifResult
		out.RuleID = f.RuleID
		out.Level = sarifLevel(f.Severity)
		out.Message.Text = fmt.Sprintf("%s — %s", f.Title, f.Why)
		if f.Snippet != "" {
			out.Message.Text += " | " + f.Snippet
		}
		var loc struct {
			PhysicalLocation struct {
				ArtifactLocation struct {
					Uri string `json:"uri"`
				} `json:"artifactLocation"`
				Region struct {
					StartLine int `json:"startLine"`
				} `json:"region"`
			} `json:"physicalLocation"`
		}
		loc.PhysicalLocation.ArtifactLocation.Uri = f.File
		loc.PhysicalLocation.Region.StartLine = f.Line
		out.Locations = append(out.Locations, loc)
		doc.Runs[0].Results = append(doc.Runs[0].Results, out)
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// RenderTerminal prints a human summary: score banner plus findings grouped
// by severity (critical first).
func RenderTerminal(w io.Writer, res *Result) {
	fmt.Fprintf(w, "verax scan: %s — score %d/100 (%d findings, %d files)\n\n",
		res.Target, res.Score(), len(res.Findings), res.Files)

	if len(res.Findings) == 0 {
		fmt.Fprintln(w, "no findings — clean. / 未发现问题。")
		return
	}

	ordered := append([]Finding(nil), res.Findings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Severity.Weight() > ordered[j].Severity.Weight()
	})

	bySev := map[Severity][]Finding{}
	for _, f := range ordered {
		bySev[f.Severity] = append(bySev[f.Severity], f)
	}
	for _, sev := range []Severity{SevCritical, SevHigh, SevMedium, SevLow} {
		fs := bySev[sev]
		if len(fs) == 0 {
			continue
		}
		fmt.Fprintf(w, "[%s] %d\n", strings.ToUpper(string(sev)), len(fs))
		for _, f := range fs {
			fmt.Fprintf(w, "  %s\n", f.Reason())
			if f.Snippet != "" {
				fmt.Fprintf(w, "      %s\n", f.Snippet)
			}
			fmt.Fprintf(w, "      → %s\n", f.Why)
		}
		fmt.Fprintln(w)
	}
}
