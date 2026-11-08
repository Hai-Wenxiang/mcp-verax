// Package report aggregates registry pulse JSON files into an ecosystem
// report (markdown + self-contained HTML): score distribution, finding
// severity mix, most prevalent rules, riskiest packages, and per-package
// trend. Data grows automatically with each nightly/weekly pulse.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Finding mirrors one scanner finding inside a pulse row (PascalCase keys:
// scanner.Finding has no json tags, so the pulse JSON carries field names).
type Finding struct {
	RuleID   string `json:"RuleID"`
	Title    string `json:"Title"`
	Severity string `json:"Severity"`
}

// Row is one scanned package in a pulse.
type Row struct {
	Name     string    `json:"name"`
	Package  string    `json:"package"`
	Score    int       `json:"score"`
	Findings []Finding `json:"findings"`
	Error    string    `json:"error"`
}

// Pulse mirrors cmd/verax registry-pulse output (the subset we aggregate).
type Pulse struct {
	Date string `json:"date"`
	Rows []Row  `json:"rows"`
}

// RuleStat is one rule's prevalence across all pulses.
type RuleStat struct {
	ID       string
	Title    string
	Severity string
	Count    int
}

// PkgStat is a package's latest-pulse snapshot.
type PkgStat struct {
	Name     string
	Package  string
	Score    int
	Critical int
	High     int
}

// TrendStat is a package's score movement between its first and last pulse.
type TrendStat struct {
	Name  string
	Pkg   string
	First int
	Last  int
	Delta int
	Dates int
}

// pkgBase strips a trailing @version from an npm spec ("risky@2.0.0" →
// "risky", "@scope/name@1.0.0" → "@scope/name") so a package's history
// survives version bumps across pulses.
func pkgBase(spec string) string {
	if i := strings.LastIndex(spec, "@"); i > 0 {
		return spec[:i]
	}
	return spec
}

// Report is the aggregated ecosystem snapshot.
type Report struct {
	AsOf          string // newest pulse date
	FirstDate     string // oldest pulse date
	Pulses        int
	Packages      int // unique packages with a successful latest scan
	MeanScore     float64
	Bands         map[string]int
	BandOrder     []string
	SevCounts     map[string]int // across all pulses
	TotalFindings int
	TopRules      []RuleStat
	Riskiest      []PkgStat
	Trend         []TrendStat
}

var bandDefs = []struct {
	Label string
	Min   int
}{
	{"90-100", 90}, {"80-89", 80}, {"60-79", 60}, {"0-59", 0},
}

// Aggregate folds pulses (any order) into a Report. "Latest" per package is
// the row from the greatest pulse date; rule prevalence counts every pulse.
func Aggregate(pulses []Pulse) *Report {
	sort.Slice(pulses, func(i, j int) bool { return pulses[i].Date < pulses[j].Date })
	r := &Report{
		Bands:     map[string]int{},
		SevCounts: map[string]int{},
	}
	for _, b := range bandDefs {
		r.BandOrder = append(r.BandOrder, b.Label)
	}
	if len(pulses) == 0 {
		return r
	}
	r.Pulses = len(pulses)
	r.FirstDate, r.AsOf = pulses[0].Date, pulses[len(pulses)-1].Date

	ruleCount := map[string]*RuleStat{}
	firstSeen := map[string]string{}          // pkg base -> first date with a score
	firstScore := map[string]int{}            // pkg base -> score at firstSeen
	seenDates := map[string]map[string]bool{} // pkg base -> distinct pulse dates
	latest := map[string]Row{}                // pkg base -> newest successful row

	for _, p := range pulses {
		for _, row := range p.Rows {
			if row.Error != "" {
				continue
			}
			key := pkgBase(row.Package)
			latest[key] = row
			if _, ok := firstSeen[key]; !ok {
				firstSeen[key] = p.Date
				firstScore[key] = row.Score
			}
			if seenDates[key] == nil {
				seenDates[key] = map[string]bool{}
			}
			seenDates[key][p.Date] = true
			for _, f := range row.Findings {
				r.SevCounts[f.Severity]++
				r.TotalFindings++
				key := f.RuleID
				st, ok := ruleCount[key]
				if !ok {
					st = &RuleStat{ID: f.RuleID, Title: f.Title, Severity: f.Severity}
					ruleCount[key] = st
				}
				st.Count++
			}
		}
	}

	sum := 0
	for _, row := range latest {
		r.Packages++
		sum += row.Score
		for _, b := range bandDefs {
			if row.Score >= b.Min {
				r.Bands[b.Label]++
				break
			}
		}
	}
	if r.Packages > 0 {
		r.MeanScore = float64(sum) / float64(r.Packages)
	}

	for _, st := range ruleCount {
		r.TopRules = append(r.TopRules, *st)
	}
	sort.Slice(r.TopRules, func(i, j int) bool {
		if r.TopRules[i].Count != r.TopRules[j].Count {
			return r.TopRules[i].Count > r.TopRules[j].Count
		}
		return r.TopRules[i].ID < r.TopRules[j].ID
	})
	if len(r.TopRules) > 10 {
		r.TopRules = r.TopRules[:10]
	}

	for _, row := range latest {
		ps := PkgStat{Name: row.Name, Package: row.Package, Score: row.Score}
		for _, f := range row.Findings {
			switch f.Severity {
			case "critical":
				ps.Critical++
			case "high":
				ps.High++
			}
		}
		r.Riskiest = append(r.Riskiest, ps)
	}
	sort.Slice(r.Riskiest, func(i, j int) bool {
		if r.Riskiest[i].Score != r.Riskiest[j].Score {
			return r.Riskiest[i].Score < r.Riskiest[j].Score
		}
		sev := func(p PkgStat) int { return p.Critical*10 + p.High }
		return sev(r.Riskiest[i]) > sev(r.Riskiest[j])
	})
	if len(r.Riskiest) > 10 {
		r.Riskiest = r.Riskiest[:10]
	}

	for pkg, row := range latest {
		if len(seenDates[pkg]) < 2 {
			continue // single pulse: no movement yet
		}
		r.Trend = append(r.Trend, TrendStat{
			Name: row.Name, Pkg: pkg,
			First: firstScore[pkg], Last: row.Score,
			Delta: row.Score - firstScore[pkg], Dates: len(seenDates[pkg]),
		})
	}
	sort.Slice(r.Trend, func(i, j int) bool { return r.Trend[i].Delta < r.Trend[j].Delta })
	if len(r.Trend) > 10 {
		r.Trend = r.Trend[:10]
	}
	return r
}

// span renders the pulse-date span ("N pulses · A→B" or just the date).
func (r *Report) span() string {
	if r.Pulses <= 1 {
		return fmt.Sprintf("%d pulse", r.Pulses)
	}
	return fmt.Sprintf("%d pulses · %s → %s", r.Pulses, r.FirstDate, r.AsOf)
}

// RenderMarkdown writes the report as GitHub-flavored markdown.
func (r *Report) RenderMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Verax Ecosystem Report\n\n")
	fmt.Fprintf(&b, "_As of %s · %s · %d unique packages · mean score %.1f_\n\n", r.AsOf, r.span(), r.Packages, r.MeanScore)

	fmt.Fprintf(&b, "## Score distribution (latest pulse per package)\n\n| Band | Packages |\n|---|---|\n")
	for _, label := range r.BandOrder {
		fmt.Fprintf(&b, "| %s | %d |\n", label, r.Bands[label])
	}

	fmt.Fprintf(&b, "\n## Findings by severity (all pulses)\n\n")
	fmt.Fprintf(&b, "critical **%d** · high **%d** · medium **%d** · low **%d** · total **%d**\n",
		r.SevCounts["critical"], r.SevCounts["high"], r.SevCounts["medium"], r.SevCounts["low"], r.TotalFindings)

	fmt.Fprintf(&b, "\n## Most prevalent rules\n\n| Rule | Severity | Hits |\n|---|---|---|\n")
	for _, st := range r.TopRules {
		fmt.Fprintf(&b, "| `%s` %s | %s | %d |\n", st.ID, st.Title, st.Severity, st.Count)
	}

	fmt.Fprintf(&b, "\n## Riskiest packages\n\n| Package | Score | crit/high |\n|---|---|---|\n")
	for _, ps := range r.Riskiest {
		fmt.Fprintf(&b, "| `%s` | %d | %d/%d |\n", ps.Package, ps.Score, ps.Critical, ps.High)
	}

	fmt.Fprintf(&b, "\n## Trend (packages with ≥2 pulses)\n\n")
	if len(r.Trend) == 0 {
		fmt.Fprintf(&b, "_Needs more nightly pulses — movement appears from the second pulse date._\n")
	} else {
		fmt.Fprintf(&b, "| Package | first → last | Δ |\n|---|---|---|\n")
		for _, t := range r.Trend {
			fmt.Fprintf(&b, "| `%s` | %d → %d | %+d |\n", t.Pkg, t.First, t.Last, t.Delta)
		}
	}
	fmt.Fprintf(&b, "\n_Generated by `verax report`; source pulses in `docs/registry/`._\n")
	return b.String()
}

// GenerateDir loads every pulse-*.json in dir (excluding pulse-latest.json),
// aggregates them and writes report.md + report.html next to the pulses.
// Returns the number of pulses folded in.
func GenerateDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "pulse-") && strings.HasSuffix(e.Name(), ".json") && e.Name() != "pulse-latest.json" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var pulses []Pulse
	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return 0, err
		}
		var p Pulse
		if err := json.Unmarshal(raw, &p); err != nil {
			continue // a malformed pulse shouldn't sink the report
		}
		pulses = append(pulses, p)
	}
	if len(pulses) == 0 {
		return 0, fmt.Errorf("no pulse-*.json found in %s", dir)
	}
	r := Aggregate(pulses)
	if err := os.WriteFile(filepath.Join(dir, "report.md"), []byte(r.RenderMarkdown()), 0o644); err != nil {
		return 0, err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.html"), []byte(r.RenderHTML()), 0o644); err != nil {
		return 0, err
	}
	return len(pulses), nil
}
