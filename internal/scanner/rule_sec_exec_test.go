package scanner

import "testing"

func TestRuleEnvDump(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-SEC-001")
	if len(got) != 2 {
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 env-dump findings (stringify + spread), got %d", len(got))
	}
	for _, f := range got {
		if f.Severity != SevCritical {
			t.Errorf("env dump must be critical: %+v", f)
		}
		if f.File != "envdump.js" {
			t.Errorf("wrong file: %+v", f)
		}
	}
}

func TestRuleShellExec(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	var got []Finding
	for _, f := range findingsBy(res, "VX-SEC-003") {
		if f.File == "shell.js" { // python findings covered by TestPythonRules
			got = append(got, f)
		}
	}
	if len(got) != 2 { // exec line + execSync line (shell:true same line deduped)
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 shell findings, got %d", len(got))
	}
	for _, f := range got {
		if f.Severity != SevHigh {
			t.Errorf("finding wrong: %+v", f)
		}
	}
}

func TestCleanFixtureNoExecOrEnvFindings(t *testing.T) {
	res, err := Scan(fixtureDir("scanclean"), "scanclean-fixture")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"VX-SEC-001", "VX-SEC-003"} {
		if n := len(findingsBy(res, id)); n != 0 {
			t.Errorf("clean fixture has %d %s findings", n, id)
		}
	}
}
