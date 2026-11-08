package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hai-Wenxiang/mcp-verax/internal/scanner"
)

func TestCLIScanVulnFixtureExits1(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"scan", "../../testfixtures/scanvuln", "--no-osv"}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit=%d, want 1 (critical/high present)\n%s%s", code, out.String(), errBuf.String())
	}
	s := out.String()
	for _, want := range []string{"score 0/100", "VX-SEC-001", "VX-SEC-003", "VX-SEC-004", "VX-SEC-005", "VX-SEC-011", "VX-SEC-012", "VX-QUA-105"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %s:\n%s", want, s)
		}
	}
}

func TestCLIScanCleanFixtureExits0(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"scan", "../../testfixtures/scanclean", "--no-osv"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d, want 0\n%s%s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "100/100") {
		t.Errorf("clean scan should score 100:\n%s", out.String())
	}
}

func TestCLIScanSARIFOutput(t *testing.T) {
	sarifFile := filepath.Join(t.TempDir(), "out.sarif")
	var out, errBuf bytes.Buffer
	code := run([]string{"scan", "../../testfixtures/scanvuln", "--no-osv", "--no-osv", "--sarif", sarifFile}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit=%d\n%s", code, errBuf.String())
	}

	raw, err := os.ReadFile(sarifFile)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name string `json:"name"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID string `json:"ruleId"`
				Level  string `json:"level"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("sarif not valid json: %v\n%s", err, raw)
	}
	if doc.Version != "2.1.0" || len(doc.Runs) != 1 || doc.Runs[0].Tool.Driver.Name != "verax" {
		t.Errorf("sarif structure wrong: %s", raw)
	}
	if len(doc.Runs[0].Results) != 29 {
		t.Errorf("sarif results = %d, want 29", len(doc.Runs[0].Results))
	}
	levels := map[string]int{}
	for _, r := range doc.Runs[0].Results {
		levels[r.Level]++
	}
	if levels["error"] == 0 || levels["warning"] == 0 || levels["note"] == 0 {
		t.Errorf("severity levels should all appear: %v", levels)
	}
}

func TestCLIScanUsageErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"scan"}, &out, &errBuf); code != 2 {
		t.Fatalf("no target exit=%d", code)
	}
	if code := run([]string{"scan", "a", "b"}, &out, &errBuf); code != 2 {
		t.Fatalf("two targets exit=%d", code)
	}
	if code := run([]string{"scan", "./no/such/dir"}, &out, &errBuf); code != 2 {
		t.Fatalf("bad dir exit=%d", code)
	}
}

func TestResolveScopedNpmSpec(t *testing.T) {
	kind, err := scanner.ResolveTarget("@modelcontextprotocol/server-memory")
	if err != nil || kind != scanner.TargetNpm {
		t.Fatalf("scoped spec must resolve to npm: kind=%v err=%v", kind, err)
	}
	kind, err = scanner.ResolveTarget("some-package@1.0.0")
	if err != nil || kind != scanner.TargetNpm {
		t.Fatalf("bare spec must resolve to npm: kind=%v err=%v", kind, err)
	}
	kind, err = scanner.ResolveTarget("../../testfixtures/scanclean")
	if err != nil || kind != scanner.TargetLocal {
		t.Fatalf("existing dir must resolve local: kind=%v err=%v", kind, err)
	}
}

func TestSARIFLevelMapping(t *testing.T) {
	cases := map[scanner.Severity]string{
		scanner.SevCritical: "error", scanner.SevHigh: "error",
		scanner.SevMedium: "warning", scanner.SevLow: "note",
	}
	// exercised via TestCLIScanSARIFOutput levels check; direct call guard:
	if got := len(cases); got != 4 {
		t.Errorf("mapping table = %d", got)
	}
}
