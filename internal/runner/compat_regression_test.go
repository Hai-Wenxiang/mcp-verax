package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Bug fix regression: servers without the prompts capability (like the
// official filesystem server) must fail prompts assertions (FAIL), not blow
// up discovery (ERROR). Found while running the official compat suites.
func TestDiscoveryToleratesMissingPrompts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noprompts.suite.yaml")
	body := fmt.Sprintf(`
suite: noprompts
target:
  command: %q
  env: {MCP_NO_PROMPTS: "1"}
  timeout_startup_ms: 60000
case:
  - name: tools still discovered
    expect:
      tools.list.includes: [echo]
  - name: prompts assertion fails cleanly
    expect:
      prompts.list.includes: [greet]
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if len(report.Cases) != 2 {
		t.Fatalf("cases = %d, want 2: %+v", len(report.Cases), report.Cases)
	}
	if report.Cases[0].Status != StatusPass {
		t.Errorf("tools case: %s err=%q checks=%+v", report.Cases[0].Status, report.Cases[0].Err, report.Cases[0].Checks)
	}
	if report.Cases[1].Status != StatusFail {
		t.Errorf("prompts case should FAIL via assertion, got %s (err=%q)", report.Cases[1].Status, report.Cases[1].Err)
	}
	if report.Cases[1].Err != "" {
		t.Errorf("no infrastructure error expected: %q", report.Cases[1].Err)
	}
}

// Bug fix regression: handshake failures must carry the server's stderr tail
// (a server that dies during initialize leaves its reason on stderr).
func TestHandshakeErrorCarriesStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dead.suite.yaml")
	body := fmt.Sprintf(`
suite: dead
target:
  command: %q
  env: {MCP_DROP_HANDSHAKE: "1"}
  timeout_startup_ms: 60000
case:
  - name: anything
    expect: {tools.list.includes: [x]}
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if report.Pass() {
		t.Fatal("dropped handshake must fail the suite")
	}
	errText := report.Cases[0].Err
	if !strings.Contains(errText, "server stderr") || !strings.Contains(errText, "MCP_DROP_HANDSHAKE") {
		t.Errorf("error should include server stderr tail:\n%s", errText)
	}
}
