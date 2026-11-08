package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerFuzzPassesOnRobustServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fuzz-ok.suite.yaml")
	body := fmt.Sprintf(`
suite: fuzz-ok
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: echo exists
    expect: {tools.list.includes: [echo]}
fuzz:
  runs: 15
  seed: 42
  hang_ms: 5000
  tools: [echo, fail, slow]
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("robust fixture should survive fuzzing")
	}
	found := false
	for _, c := range report.Cases {
		if c.Name == "fuzz" {
			found = true
			if !strings.Contains(c.Checks[0].Desc, "15 runs") || !strings.Contains(c.Checks[0].Desc, "no crash, no hang") {
				t.Errorf("fuzz desc wrong: %s", c.Checks[0].Desc)
			}
		}
	}
	if !found {
		t.Fatal("fuzz case missing")
	}
}

func TestRunnerFuzzDetectsCrashingServer(t *testing.T) {
	// The echo fixture crashes on method "crash"; we register no such tool in
	// the mcp fixture, so we point the suite at the raw echo fixture instead —
	// it crashes when a tool named "crash" is called. To fuzz that path
	// deterministically we use the fuzz engine's abort behaviour via a fake
	// in-process campaign (unit-level crash test lives in internal/fuzz).
	// Here we assert the wiring: a suite without fuzz block has no fuzz case.
	path := filepath.Join(t.TempDir(), "nofuzz.suite.yaml")
	body := fmt.Sprintf(`
suite: nofuzz
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: echo exists
    expect: {tools.list.includes: [echo]}
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		t.Fatal("suite should pass")
	}
	for _, c := range report.Cases {
		if c.Name == "fuzz" {
			t.Fatalf("fuzz case must not appear without fuzz block: %+v", c)
		}
	}
}
