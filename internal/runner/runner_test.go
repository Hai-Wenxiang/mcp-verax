package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureCmd = "go run github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"

func writeSuite(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunnerHappyPath(t *testing.T) {
	path := writeSuite(t, "happy.suite.yaml", fmt.Sprintf(`
suite: happy
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: discovery
    expect:
      tools.list.includes: [echo, fail, slow, env]
      tools.list.count: 4
      tools.list.schema_valid: true
      prompts.list.includes: [greet]
      resources.list.min: 1
  - name: echo
    call: echo
    args: {msg: "hello runner"}
    expect:
      ok: true
      result.contains: ["hello runner"]
  - name: expected tool error
    call: fail
    expect:
      is_error: true
      error.contains: "on purpose"
`, fixtureCmd))

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("suite should pass")
	}
	if len(report.Cases) != 3 {
		t.Fatalf("cases = %d", len(report.Cases))
	}
	for _, c := range report.Cases {
		if c.Status != StatusPass || c.DurationMS < 0 {
			t.Errorf("case %s: status=%s dur=%d", c.Name, c.Status, c.DurationMS)
		}
	}
	if report.SnapshotFile == "" {
		t.Error("snapshot file should be reported")
	}
}

func TestRunnerAssertionFailures(t *testing.T) {
	path := writeSuite(t, "fails.suite.yaml", fmt.Sprintf(`
suite: fails
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: missing tool expected
    expect:
      tools.list.includes: [does_not_exist]
  - name: ok expected but tool errors
    call: fail
    expect:
      ok: true
`, fixtureCmd))

	report := RunSuiteFile(path, Options{})
	if report.Pass() {
		t.Fatal("suite must fail")
	}
	if report.Cases[0].Status != StatusFail || len(report.Cases[0].Checks) != 1 {
		t.Errorf("case0: %+v", report.Cases[0])
	}
	if report.Cases[1].Status != StatusFail {
		t.Errorf("case1 should fail via ok-check: %+v", report.Cases[1])
	}
}

func TestRunnerPerCaseEnvRespawn(t *testing.T) {
	path := writeSuite(t, "env.suite.yaml", fmt.Sprintf(`
suite: env-injection
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: probe A
    env: {VERAX_PROBE: alpha}
    call: env
    expect:
      ok: true
      result.contains: ["VERAX_PROBE=alpha"]
  - name: probe B
    env: {VERAX_PROBE: beta}
    call: env
    expect:
      ok: true
      result.contains: ["VERAX_PROBE=beta"]
`, fixtureCmd))

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("per-case env injection should pass (respawn mode)")
	}
}

func TestRunnerSnapshotLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.suite.yaml")
	base := fmt.Sprintf(`
suite: snap
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: stable
    call: echo
    args: {msg: %%q}
    expect:
      result.snapshot: echo-snap
`, fixtureCmd)

	write := func(msg string) {
		if err := os.WriteFile(path, []byte(fmt.Sprintf(base, msg)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// First run: snapshot recorded, passes.
	write("stable message")
	first := RunSuiteFile(path, Options{})
	if !first.Pass() {
		t.Fatalf("first run should record snapshot: %+v", first.Cases)
	}
	if _, err := os.Stat(filepath.Join(dir, "__snapshots__", "snap.suite.snap.json")); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}

	// Second run, identical args: passes against stored snapshot.
	if second := RunSuiteFile(path, Options{}); !second.Pass() {
		t.Fatalf("second run should match snapshot: %+v", second.Cases)
	}

	// Third run, changed args: snapshot mismatch fails the case.
	write("CHANGED MESSAGE")
	third := RunSuiteFile(path, Options{})
	if third.Pass() {
		t.Fatal("changed result must fail snapshot check")
	}
	found := false
	for _, ck := range third.Cases[0].Checks {
		if strings.Contains(ck.Desc, "result.snapshot") {
			found = true
			if ck.Pass || !strings.Contains(ck.Detail, "first diff") {
				t.Errorf("snapshot check should fail with diff: %+v", ck)
			}
		}
	}
	if !found {
		t.Error("snapshot check missing from results")
	}

	// Update mode accepts the new result.
	fourth := RunSuiteFile(path, Options{UpdateSnapshots: true})
	if !fourth.Pass() {
		t.Fatalf("update mode should accept new snapshot: %+v", fourth.Cases)
	}
	if fifth := RunSuiteFile(path, Options{}); !fifth.Pass() {
		t.Fatal("post-update run should pass")
	}
}

func TestRunnerServerStartFailure(t *testing.T) {
	path := writeSuite(t, "bad-target.suite.yaml", `
suite: bad-target
target:
  command: go run ./no/such/package
  timeout_startup_ms: 20000
case:
  - name: anything
    expect: {tools.list.includes: [x]}
`)
	report := RunSuiteFile(path, Options{})
	if report.Pass() {
		t.Fatal("unstartable server must fail the suite")
	}
	if report.Cases[0].Status != StatusError || !strings.Contains(report.Cases[0].Err, "start server") {
		t.Errorf("expected start error: %+v", report.Cases[0])
	}
}

func TestRunnerInvalidSuiteFile(t *testing.T) {
	report := RunSuiteFile(filepath.Join(t.TempDir(), "missing.suite.yaml"), Options{})
	if report.Pass() || report.Cases[0].Status != StatusError {
		t.Fatalf("missing file must be a failed report: %+v", report.Cases)
	}

	path := writeSuite(t, "broken.suite.yaml", "suite: [unclosed\n  bad")
	report2 := RunSuiteFile(path, Options{})
	if report2.Pass() {
		t.Fatal("invalid YAML must fail")
	}
}
