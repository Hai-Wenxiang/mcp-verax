package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGeneratesWorkingSuite(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "my.suite.yaml")

	var stdout, stderr bytes.Buffer
	code := run([]string{"init", out, "--", "go", "run", "github.com/Hai-Wenxiang/mcp-verax/testfixtures/mcp"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d\n%s%s", code, stdout.String(), stderr.String())
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{
		"suite: my",
		"tools.list.includes: [cache_dump, echo, env, fail, fetch_url, slow]",
		"tools.list.count: 6",
		"- name: call echo",
		`msg: "TODO"`,
		"leak:",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("generated suite missing %q:\n%s", want, s)
		}
	}

	// the generated suite must actually pass its discovery case
	var tout, terr bytes.Buffer
	tcode := run([]string{"test", out}, &tout, &terr)
	if tcode != 0 {
		t.Fatalf("generated suite failed (exit=%d):\n%s%s", tcode, tout.String(), terr.String())
	}
}

func TestInitUsageErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"init"}, &out, &errBuf); code != 2 {
		t.Fatalf("no args exit=%d", code)
	}
	if code := run([]string{"init", "x.yaml"}, &out, &errBuf); code != 2 {
		t.Fatalf("missing command exit=%d", code)
	}
}
