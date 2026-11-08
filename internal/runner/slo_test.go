package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSLOHappyBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slo-ok.suite.yaml")
	body := fmt.Sprintf(`
suite: slo-ok
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: echo exists
    expect: {tools.list.includes: [echo]}
slo:
  tool: echo
  args: {msg: "slo sample"}
  p95_call_ms: 10000
  concurrency: 4
  sample_calls: 8
  success_rate: 1.0
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("SLO suite should pass")
	}
	var sloCase *CaseResult
	for i := range report.Cases {
		if strings.HasPrefix(report.Cases[i].Name, "slo(") {
			sloCase = &report.Cases[i]
		}
	}
	if sloCase == nil {
		t.Fatal("slo case missing from report")
	}
	if !strings.Contains(sloCase.Checks[0].Desc, "8 samples") || !strings.Contains(sloCase.Checks[0].Desc, "p95=") {
		t.Errorf("slo check should report samples and percentiles: %s", sloCase.Checks[0].Desc)
	}
}

func TestSLOSuccessRateViolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slo-bad.suite.yaml")
	body := fmt.Sprintf(`
suite: slo-bad
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: fail tool exists
    expect: {tools.list.includes: [fail]}
slo:
  tool: fail
  sample_calls: 4
  success_rate: 1.0
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if report.Pass() {
		t.Fatal("SLO with always-failing tool must violate success_rate")
	}
	found := false
	for _, c := range report.Cases {
		if strings.HasPrefix(c.Name, "slo(") && c.Status == StatusFail {
			found = true
			if !strings.Contains(c.Checks[0].Detail, "success rate") || !strings.Contains(c.Checks[0].Detail, "below budget") {
				t.Errorf("violation detail wrong: %s", c.Checks[0].Detail)
			}
		}
	}
	if !found {
		t.Fatalf("expected failing slo case: %+v", report.Cases)
	}
}

func TestP95Index(t *testing.T) {
	cases := []struct {
		n, want int
	}{
		{1, 0}, {2, 1}, {10, 9}, {20, 19}, {100, 95}, {40, 38},
	}
	for _, c := range cases {
		if got := p95Index(c.n); got != c.want {
			t.Errorf("p95Index(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}
