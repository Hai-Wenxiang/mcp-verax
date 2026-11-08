package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoModFindings(t *testing.T) {
	dir := t.TempDir()
	gomod := "module example.com/vuln-go\n" // no go directive; no LICENSE/README
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Scan(dir, "vuln-go")
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, f := range res.Findings {
		if f.RuleID == "VX-QUA-105" || f.RuleID == "VX-QUA-106" {
			titles = append(titles, f.Title)
		}
	}
	joined := titles
	wantAny := map[string]bool{"go.mod has no go directive": false, "no LICENSE file": false, "no README": false}
	for _, t2 := range joined {
		if _, ok := wantAny[t2]; ok {
			wantAny[t2] = true
		}
	}
	for want, seen := range wantAny {
		if !seen {
			t.Errorf("missing finding %q; got %v", want, joined)
		}
	}
}

func TestScanOwnRepoGoMod(t *testing.T) {
	// our own repo is a well-formed Go project: must scan without manifest errors
	res, err := Scan("../..", "mcp-verax")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Findings {
		if f.Title == "go.mod has no module directive" || f.Title == "go.mod has no go directive" ||
			f.Title == "no LICENSE file" || f.Title == "no README" {
			t.Errorf("own repo flagged: %s", f.Reason())
		}
	}
}
