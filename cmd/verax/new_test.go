package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	verax "github.com/Hai-Wenxiang/mcp-verax"
)

func TestNewScaffoldsWorkingProject(t *testing.T) {
	if nodeAvailable() == false {
		t.Skip("node not available")
	}
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	code := run([]string{"new", "csv-query", "my-csv", "--dir", dir}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit=%d\n%s%s", code, out.String(), errBuf.String())
	}

	proj := filepath.Join(dir, "my-csv")
	for _, f := range []string{"index.js", "package.json", "README.md", "tests/server.suite.yaml", "data/people.csv"} {
		if _, err := os.Stat(filepath.Join(proj, filepath.FromSlash(f))); err != nil {
			t.Errorf("missing scaffolded file %s: %v", f, err)
		}
	}

	// package renamed
	raw, _ := os.ReadFile(filepath.Join(proj, "package.json"))
	if !strings.Contains(string(raw), `"name": "my-csv"`) {
		t.Errorf("package not renamed: %s", raw)
	}

	// suite localized: command/env rewritten to scaffold-local paths
	suite, _ := os.ReadFile(filepath.Join(proj, "tests", "server.suite.yaml"))
	if !strings.Contains(string(suite), "command: node index.js") ||
		!strings.Contains(string(suite), "VERAX_ROOT: ./data") ||
		strings.Contains(string(suite), "command: node templates/") {
		t.Errorf("suite command not localized:\n%s", suite)
	}

	// scaffolded suite actually passes when run from the project dir
	// (the documented usage), exercising the whole flywheel from the scaffold
	wd, _ := os.Getwd()
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	var tout, terr bytes.Buffer
	tcode := run([]string{"test", "tests/server.suite.yaml"}, &tout, &terr)
	if tcode != 0 {
		t.Fatalf("scaffolded suite failed (exit=%d):\n%s%s", tcode, tout.String(), terr.String())
	}
}

func TestNewErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := run([]string{"new"}, &out, &errBuf); code != 2 {
		t.Fatalf("no args exit=%d", code)
	}
	if code := run([]string{"new", "no-such-tmpl", "x", "--dir", t.TempDir()}, &out, &errBuf); code != 2 {
		t.Fatalf("unknown template exit=%d", code)
	}
	if code := run([]string{"new", "csv-query", "bad name!", "--dir", t.TempDir()}, &out, &errBuf); code != 2 {
		t.Fatalf("bad name exit=%d", code)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "exists"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"new", "csv-query", "exists", "--dir", dir}, &out, &errBuf); code != 1 {
		t.Fatalf("existing dest exit=%d", code)
	}
}

func TestEmbeddedTemplateRegistry(t *testing.T) {
	ids := verax.TemplateIDs()
	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found["filesystem-safe"] || !found["csv-query"] {
		t.Errorf("embedded templates missing: %v", ids)
	}
}

func nodeAvailable() bool {
	return true // node v24 verified in dev; runners carry node
}
