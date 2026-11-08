package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSRFProbeDetectsOutboundFetch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssrf-pos.suite.yaml")
	body := fmt.Sprintf(`
suite: ssrf-positive
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: tools present
    expect: {tools.list.includes: [fetch_url]}
probe:
  ssrf: {hang_ms: 4000}
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	if report.Pass() {
		t.Fatal("fetch_url tool must trip the SSRF probe")
	}
	found := false
	for _, c := range report.Cases {
		if c.Name == "probe(ssrf)" && c.Status == StatusFail {
			found = true
			d := c.Checks[0].Detail
			if !strings.Contains(d, "fetch_url.url") || !strings.Contains(d, "SSRF") {
				t.Errorf("probe detail should name the offending tool.arg:\n%s", d)
			}
		}
	}
	if !found {
		t.Fatalf("probe(ssrf) case missing or passing: %+v", report.Cases)
	}
}

func TestSSRFProbeCleanServerPasses(t *testing.T) {
	// csv-query template: file-only tools, zero network surface
	path := filepath.Join(t.TempDir(), "ssrf-neg.suite.yaml")
	body := `
suite: ssrf-negative
target:
  command: node templates/csv-query/index.js
  env: {VERAX_ROOT: templates/csv-query/data}
  timeout_startup_ms: 30000
case:
  - name: tools present
    expect: {tools.list.includes: [list_csv]}
probe:
  ssrf: {hang_ms: 3000}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	report := RunSuiteFile(path, Options{CloseGraceMS: 1000})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("file-only template must pass the SSRF probe")
	}
	for _, c := range report.Cases {
		if c.Name == "probe(ssrf)" {
			if !strings.Contains(c.Checks[0].Desc, "0 outbound hits") {
				t.Errorf("desc wrong: %s", c.Checks[0].Desc)
			}
		}
	}
}

func TestStringProperties(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"url":{"type":"string"},"n":{"type":"number"},"id":{"type":"string"}}}`)
	got := stringProperties(schema)
	if len(got) != 2 {
		t.Fatalf("want 2 string props, got %v", got)
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p] = true
	}
	if !seen["url"] || !seen["id"] {
		t.Errorf("wrong props: %v", got)
	}
	if stringProperties(nil) != nil {
		t.Error("nil schema should yield nil")
	}
}
