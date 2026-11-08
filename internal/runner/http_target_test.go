package runner

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func startHTTPFixture(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	cmd := exec.Command("go", "run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp")
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("MCP_HTTP_PORT=%d", port))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
			_ = c.Close()
			return fmt.Sprintf("http://%s", addr)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("fixture http not ready")
	return ""
}

func TestRunnerHTTPTargetSuite(t *testing.T) {
	base := startHTTPFixture(t)
	path := filepath.Join(t.TempDir(), "http.suite.yaml")
	body := fmt.Sprintf(`
suite: http-target
target:
  url: %q
  timeout_startup_ms: 60000
case:
  - name: discovery over http
    expect:
      tools.list.includes: [echo, fail, slow, env]
      tools.list.count: 5
      prompts.list.includes: [greet]
  - name: echo over http
    call: echo
    args: {msg: "http round trip"}
    expect:
      ok: true
      result.contains: ["http round trip"]
  - name: tool error over http
    call: fail
    expect:
      is_error: true
slo:
  tool: echo
  args: {msg: "s"}
  p95_call_ms: 30000
  sample_calls: 5
fuzz:
  runs: 5
  seed: 3
  hang_ms: 30000
  tools: [echo]
`, base)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("http-target suite should pass end to end")
	}
	if len(report.Cases) != 5 { // 3 cases + slo + fuzz
		t.Errorf("cases = %d, want 5", len(report.Cases))
	}
}

func TestRunnerHTTPTargetBadHostFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.suite.yaml")
	body := `
suite: http-bad
target:
  url: "http://127.0.0.1:1"
  timeout_startup_ms: 3000
case:
  - name: anything
    expect: {tools.list.includes: [x]}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := RunSuiteFile(path, Options{})
	if report.Pass() || report.Cases[0].Status != StatusError {
		t.Fatalf("unreachable http target must error: %+v", report.Cases)
	}
}
