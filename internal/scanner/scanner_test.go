package scanner

import (
	"path/filepath"
	"strings"
	"testing"
)

func fixtureDir(name string) string {
	return filepath.Join("..", "..", "testfixtures", name)
}

func TestScanCleanFixture(t *testing.T) {
	res, err := Scan(fixtureDir("scanclean"), "scanclean-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 0 {
		for _, f := range res.Findings {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("clean fixture must have 0 findings, got %d", len(res.Findings))
	}
	if res.Score() != 100 {
		t.Errorf("score = %d, want 100", res.Score())
	}
}

func TestScanVulnMetadataFindings(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}

	var titles []string
	for _, f := range res.Findings {
		if f.RuleID != "VX-QUA-105" && f.RuleID != "VX-QUA-106" {
			t.Errorf("unexpected rule fired: %s", f.Reason())
		}
		titles = append(titles, f.Title)
	}
	joined := strings.Join(titles, "|")
	for _, want := range []string{"no license declared", "no repository link", "no description", "no README"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing finding %q; got %v", want, titles)
		}
	}
	// 4 x Low (1 point each) => 96
	if res.Score() != 96 {
		t.Errorf("score = %d, want 96", res.Score())
	}
}

func TestScanMissingPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := writeLine(filepath.Join(dir, "a.js"), "console.log(1)"); err != nil {
		t.Fatal(err)
	}
	res, err := Scan(dir, "no-manifest")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range res.Findings {
		if f.Title == "package.json missing" && f.Severity == SevMedium {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing manifest must be flagged medium: %+v", res.Findings)
	}
}

func TestSeverityWeights(t *testing.T) {
	cases := map[Severity]int{
		SevCritical: 25, SevHigh: 10, SevMedium: 4, SevLow: 1,
	}
	for sev, want := range cases {
		if got := sev.Weight(); got != want {
			t.Errorf("%s weight = %d, want %d", sev, got, want)
		}
	}
}

func TestScoreFloorsAtZero(t *testing.T) {
	r := &Result{Findings: []Finding{
		{Severity: SevCritical}, {Severity: SevCritical},
		{Severity: SevCritical}, {Severity: SevCritical},
		{Severity: SevCritical}, // 125 points of deductions
	}}
	if r.Score() != 0 {
		t.Errorf("score = %d, want 0 (floor)", r.Score())
	}
}

func writeLine(path, content string) error {
	return osWriteFile(path, content)
}
