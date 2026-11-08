package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeakCleanServerPasses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leak-ok.suite.yaml")
	body := fmt.Sprintf(`
suite: leak-ok
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: echo exists
    expect: {tools.list.includes: [echo]}
leak:
  process_exit: true
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("clean server should pass leak check")
	}
	found := false
	for _, c := range report.Cases {
		if c.Name == "leak(process_exit)" {
			found = true
			if !strings.Contains(c.Checks[0].Desc, "1/1 server processes exited") {
				t.Errorf("leak desc wrong: %s", c.Checks[0].Desc)
			}
		}
	}
	if !found {
		t.Fatal("leak(process_exit) case missing")
	}
}

func TestLeakDetectsHangingServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "leak-bad.suite.yaml")
	body := fmt.Sprintf(`
suite: leak-bad
target:
  command: %q
  env: {MCP_IGNORE_EOF: "1"}
  timeout_startup_ms: 60000
case:
  - name: echo exists
    expect: {tools.list.includes: [echo]}
leak:
  process_exit: true
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{CloseGraceMS: 400})
	if report.Pass() {
		t.Fatal("hanging server must fail the leak check")
	}
	found := false
	for _, c := range report.Cases {
		if c.Name == "leak(process_exit)" {
			found = true
			if c.Status != StatusFail {
				t.Errorf("leak case status = %s, want fail", c.Status)
			}
			if !strings.Contains(c.Checks[0].Detail, "leaked") || !strings.Contains(c.Checks[0].Detail, "force-killed") {
				t.Errorf("leak detail wrong: %s", c.Checks[0].Detail)
			}
		}
	}
	if !found {
		t.Fatalf("leak case missing: %+v", report.Cases)
	}
}

func TestNoLeakBlockMeansNoLeakCase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noleak.suite.yaml")
	body := fmt.Sprintf(`
suite: noleak
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
		if strings.HasPrefix(c.Name, "leak(") {
			t.Fatalf("leak case must not appear without leak block: %+v", c)
		}
	}
}
