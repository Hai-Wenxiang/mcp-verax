package scanner

import (
	"strings"
	"testing"
)

func findingsBy(res *Result, rule string) []Finding {
	var out []Finding
	for _, f := range res.Findings {
		if f.RuleID == rule {
			out = append(out, f)
		}
	}
	return out
}

func TestRuleEvalInput(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	var got []Finding
	for _, f := range findingsBy(res, "VX-SEC-004") {
		if f.File == "eval.js" { // python findings covered by TestPythonRules
			got = append(got, f)
		}
	}
	if len(got) != 2 {
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 eval findings (eval + new Function), got %d", len(got))
	}
	for _, f := range got {
		if f.Severity != SevHigh || f.Line == 0 || f.Snippet == "" {
			t.Errorf("finding must carry severity/line/snippet: %+v", f)
		}
	}
	// The commented-out eval must NOT be among them.
	for _, f := range got {
		if strings.Contains(f.Snippet, "comment") {
			t.Errorf("comment line flagged: %s", f.Snippet)
		}
	}
}

func TestRuleHardcodedSecrets(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-SEC-005")
	if len(got) < 3 {
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want >=3 secret findings (sk-, AKIA, xoxb, ghp_), got %d", len(got))
	}
	seenLines := map[int]bool{}
	for _, f := range got {
		if f.File != "secrets.js" {
			t.Errorf("unexpected file: %+v", f)
		}
		if seenLines[f.Line] {
			t.Errorf("duplicate finding on line %d", f.Line)
		}
		seenLines[f.Line] = true
	}
}

func TestCleanFixtureStaysClean(t *testing.T) {
	res, err := Scan(fixtureDir("scanclean"), "scanclean-fixture")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(findingsBy(res, "VX-SEC-004")) + len(findingsBy(res, "VX-SEC-005")); n != 0 {
		for _, f := range res.Findings {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("clean fixture must have no SEC findings, got %d", n)
	}
}

func TestVulnScoreDrops(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	// previous 20 + Python (2 eval + 2 shell + 2 deser-critical)
	// + 1 empty catch + 2 PII patterns = 29 findings, past the floor.
	if len(res.Findings) != 29 {
		t.Errorf("findings = %d, want 29", len(res.Findings))
		for _, f := range res.Findings {
			t.Logf("%s", f.Reason())
		}
	}
	if res.Score() != 0 {
		t.Errorf("score = %d, want 0 (floor)", res.Score())
	}
}
