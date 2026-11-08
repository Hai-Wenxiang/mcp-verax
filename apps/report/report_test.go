package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustPulse(t *testing.T, date string, rows []Row) string {
	t.Helper()
	raw, err := json.Marshal(Pulse{Date: date, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestAggregate(t *testing.T) {
	pulses := []Pulse{
		{Date: "2026-01-02", Rows: []Row{
			// risky improved from 40 to 60 across pulses
			{Name: "risky", Package: "risky@2.0.0", Score: 60, Findings: []Finding{
				{RuleID: "VX-SEC-005", Title: "credential literal", Severity: "high"},
			}},
			// clean stayed 100
			{Name: "clean", Package: "clean@1.0.1", Score: 100},
			// error rows don't count as scans
			{Name: "broken", Package: "broken@1.0.0", Error: "fetch failed"},
		}},
		{Date: "2026-01-01", Rows: []Row{
			{Name: "risky", Package: "risky@1.0.0", Score: 40, Findings: []Finding{
				{RuleID: "VX-SEC-005", Title: "credential literal", Severity: "high"},
				{RuleID: "VX-SEC-005", Title: "credential literal", Severity: "high"},
				{RuleID: "VX-QUA-105", Title: "missing license", Severity: "low"},
			}},
			{Name: "clean", Package: "clean@1.0.0", Score: 100},
			{Name: "gone", Package: "gone@1.0.0", Score: 90},
		}},
	}
	r := Aggregate(pulses)

	if r.AsOf != "2026-01-02" || r.FirstDate != "2026-01-01" || r.Pulses != 2 {
		t.Fatalf("span wrong: %+v", r)
	}
	// unique pkg bases: risky, clean, gone (broken errored on latest → excluded)
	if r.Packages != 3 {
		t.Fatalf("packages = %d, want 3 (latest-error rows excluded)", r.Packages)
	}
	if want := float64(60+100+90) / 3; r.MeanScore != want {
		t.Fatalf("mean = %v, want %v", r.MeanScore, want)
	}
	if r.Bands["90-100"] != 2 || r.Bands["60-79"] != 1 {
		t.Fatalf("bands = %v", r.Bands)
	}
	if r.SevCounts["high"] != 3 || r.SevCounts["low"] != 1 || r.TotalFindings != 4 {
		t.Fatalf("sev counts = %v total=%d", r.SevCounts, r.TotalFindings)
	}
	if len(r.TopRules) == 0 || r.TopRules[0].ID != "VX-SEC-005" || r.TopRules[0].Count != 3 {
		t.Fatalf("top rules = %+v", r.TopRules)
	}
	if len(r.Riskiest) != 3 || r.Riskiest[0].Package != "risky@2.0.0" || r.Riskiest[0].Score != 60 {
		t.Fatalf("riskiest = %+v", r.Riskiest)
	}
	if r.Riskiest[0].High != 1 {
		t.Fatalf("riskiest high count = %d, want 1", r.Riskiest[0].High)
	}
	// trend: version bump risky@1→risky@2 must stay one package: 40→60 (+20);
	// clean 100→100 (Δ0); gone saw only one date → excluded
	if len(r.Trend) != 2 {
		t.Fatalf("trend = %+v", r.Trend)
	}
	var riskyTrend *TrendStat
	for i := range r.Trend {
		if r.Trend[i].Pkg == "risky" {
			riskyTrend = &r.Trend[i]
		}
	}
	if riskyTrend == nil || riskyTrend.Delta != 20 || riskyTrend.First != 40 || riskyTrend.Last != 60 {
		t.Fatalf("risky trend = %+v", riskyTrend)
	}
}

func TestAggregateTrendUsesFirstSeenAcrossPulses(t *testing.T) {
	pulses := []Pulse{
		{Date: "2026-01-03", Rows: []Row{{Name: "a", Package: "a@1", Score: 70}}},
		{Date: "2026-01-01", Rows: []Row{{Name: "a", Package: "a@1", Score: 90}}},
		{Date: "2026-01-02", Rows: []Row{{Name: "a", Package: "a@1", Score: 50}}},
	}
	r := Aggregate(pulses)
	if len(r.Trend) != 1 || r.Trend[0].First != 90 || r.Trend[0].Last != 70 || r.Trend[0].Delta != -20 {
		t.Fatalf("trend = %+v", r.Trend)
	}
}

func TestRenderersAndGenerateDir(t *testing.T) {
	dir := t.TempDir()
	p1 := mustPulse(t, "2026-01-01", []Row{
		{Name: "risky", Package: "risky@1", Score: 40, Findings: []Finding{
			{RuleID: "VX-SEC-005", Title: "credential literal", Severity: "critical"},
		}},
		{Name: "clean", Package: "clean@1", Score: 100},
	})
	// decoys that must be ignored
	for name, content := range map[string]string{
		"pulse-2026-01-01.json": p1,
		"pulse-latest.json":     p1,
		"pulse-broken.json":     "{not json",
		"notes.txt":             "ignore me",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	n, err := GenerateDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("pulses folded = %d, want 1", n)
	}
	md, err := os.ReadFile(filepath.Join(dir, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Verax Ecosystem Report", "As of 2026-01-01",
		"Score distribution", "VX-SEC-005", "Riskiest packages",
		"Needs more nightly pulses",
	} {
		if !strings.Contains(string(md), want) {
			t.Errorf("markdown missing %q", want)
		}
	}
	html, err := os.ReadFile(filepath.Join(dir, "report.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<title>Verax Ecosystem Report</title>", "index.html", "sev-critical"} {
		if !strings.Contains(string(html), want) {
			t.Errorf("html missing %q", want)
		}
	}
}

func TestGenerateDirNoPulses(t *testing.T) {
	dir := t.TempDir()
	if _, err := GenerateDir(dir); err == nil {
		t.Fatal("expected error with no pulse files")
	}
}
