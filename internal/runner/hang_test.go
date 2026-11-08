package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerHangTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hang.suite.yaml")
	body := fmt.Sprintf(`
suite: hang
target:
  command: %q
  env: {MCP_SLOW_MS: "8000"}
  timeout_startup_ms: 60000
case:
  - name: slow call must time out
    call: slow
    timeout_ms: 300
    expect:
      ok: true
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{CloseGraceMS: 400})
	if report.Pass() {
		t.Fatal("hanging tool must fail the case")
	}
	c := report.Cases[0]
	if c.Status != StatusFail {
		t.Errorf("status = %s, want fail (assertion-driven)", c.Status)
	}
	if !strings.Contains(c.Checks[0].Detail, "timed out after 300ms") {
		t.Errorf("detail should name the timeout budget: %+v", c.Checks[0])
	}
	if c.DurationMS > 2000 {
		t.Errorf("case took %dms, timeout not enforced", c.DurationMS)
	}
}
