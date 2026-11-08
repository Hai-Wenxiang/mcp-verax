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

func TestEnvDefaultParamIsMediumNotCritical(t *testing.T) {
	// found via live registry pulse: `agencies(env = process.env)` DI idiom
	content := "export function agencies(env = process.env) {\n  return list;\n}\nexport function dump() {\n  return JSON.stringify(process.env);\n}\n"
	got := secEnvDump.Check("di.js", content)
	if len(got) != 2 {
		t.Fatalf("want 2 findings (DI + exposure), got %d: %+v", len(got), got)
	}
	var di, exposure *Finding
	for i := range got {
		if got[i].Title == "env dependency-injection default (review outputs)" {
			di = &got[i]
		}
		if got[i].Severity == SevCritical {
			exposure = &got[i]
		}
	}
	if di == nil || di.Severity != SevMedium {
		t.Errorf("DI default must be medium with distinct title: %+v", di)
	}
	if exposure == nil {
		t.Error("raw stringify exposure must stay critical")
	}
}
