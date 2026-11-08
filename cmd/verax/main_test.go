package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureCmd = "go run github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"

const happySuite = `
suite: cli-happy
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: discovery
    expect:
      tools.list.includes: [echo]
  - name: echo
    call: echo
    args: {msg: "via cli"}
    expect:
      ok: true
      result.contains: ["via cli"]
`

const failingSuite = `
suite: cli-fail
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: impossible
    expect:
      tools.list.includes: [no_such_tool]
`

func writeTemp(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCLITestHappyPath(t *testing.T) {
	path := writeTemp(t, "cli.suite.yaml", fmt.Sprintf(happySuite, fixtureCmd))
	var out, errBuf bytes.Buffer
	code := run([]string{"test", path}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d, output:\n%s%s", code, out.String(), errBuf.String())
	}
	if !strings.Contains(out.String(), "verax: OK — 2/2 cases passed") {
		t.Errorf("verdict missing:\n%s", out.String())
	}
}

func TestCLITestFailingSuiteExits1(t *testing.T) {
	path := writeTemp(t, "cli.suite.yaml", fmt.Sprintf(failingSuite, fixtureCmd))
	var out, errBuf bytes.Buffer
	code := run([]string{"test", path}, &out, &errBuf)
	if code != 1 {
		t.Fatalf("exit=%d, want 1", code)
	}
	if !strings.Contains(out.String(), "verax: FAILED") || !strings.Contains(out.String(), "no_such_tool") {
		t.Errorf("failure should name missing tool:\n%s", out.String())
	}
}

func TestCLITestMissingArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"test"}, &out, &errBuf); code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "at least one suite file") {
		t.Errorf("usage error missing:\n%s", errBuf.String())
	}
}

func TestCLIUnknownCommand(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"bogus"}, &out, &errBuf); code != 2 {
		t.Fatalf("exit=%d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "unknown command") {
		t.Errorf("stderr:\n%s", errBuf.String())
	}
}

func TestCLIVersion(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer
	if code := run([]string{"version"}, &out, &errBuf); code != 0 || !strings.HasPrefix(out.String(), "verax ") {
		t.Fatalf("version output wrong: %q (exit=%d)", out.String(), code)
	}
}
