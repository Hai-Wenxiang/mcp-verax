package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSideEffectProbeDetectsWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "se-pos.suite.yaml")
	body := fmt.Sprintf(`
suite: se-positive
target:
  command: %q
  timeout_startup_ms: 60000
case:
  - name: tools present
    expect: {tools.list.includes: [cache_dump]}
probe:
  side_effect:
    hang_ms: 5000
`, fixtureCmd)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{})
	found := false
	for _, c := range report.Cases {
		if c.Name == "probe(side_effect)" {
			found = true
			if c.Status != StatusFail {
				t.Fatalf("cache_dump must trip the side-effect probe: %+v", c.Checks)
			}
			d := c.Checks[0].Detail
			if !contains(d, "cache_dump") || !contains(d, "created") {
				t.Errorf("detail should name tool and change:\n%s", d)
			}
		}
	}
	if !found {
		t.Fatal("probe(side_effect) case missing")
	}
}

func TestSideEffectProbeReadOnlyServerPasses(t *testing.T) {
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	path := filepath.Join(t.TempDir(), "se-neg.suite.yaml")
	body := `
suite: se-negative
target:
  command: node templates/csv-query/index.js
  env: {VERAX_ROOT: templates/csv-query/data}
  timeout_startup_ms: 30000
case:
  - name: tools present
    expect: {tools.list.includes: [list_csv]}
probe:
  side_effect: {hang_ms: 3000}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSuiteFile(path, Options{CloseGraceMS: 1000})
	if !report.Pass() {
		for _, c := range report.Cases {
			t.Logf("%s: %s err=%q checks=%+v", c.Name, c.Status, c.Err, c.Checks)
		}
		t.Fatal("read-only template must pass the side-effect probe")
	}
	for _, c := range report.Cases {
		if c.Name == "probe(side_effect)" && !contains(c.Checks[0].Desc, "0 wrote") {
			t.Errorf("desc wrong: %s", c.Checks[0].Desc)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
