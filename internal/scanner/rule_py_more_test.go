package scanner

import "testing"

func TestPythonRules(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "v")
	if err != nil {
		t.Fatal(err)
	}
	evalN := len(findingsBy(res, "VX-SEC-004"))
	shellN := len(findingsBy(res, "VX-SEC-003"))
	deserN := len(findingsBy(res, "VX-SEC-013"))

	// eval + exec in server.py (JS eval findings live in eval.js: 2 there)
	if evalN != 4 { // 2 JS + 2 Python
		for _, f := range findingsBy(res, "VX-SEC-004") {
			t.Logf("%s", f.Reason())
		}
		t.Errorf("SEC-004 findings = %d, want 4 (2 JS + 2 Py)", evalN)
	}
	if shellN != 4 { // 2 JS shell.js + 2 Py (os.system, shell=True)
		for _, f := range findingsBy(res, "VX-SEC-003") {
			t.Logf("%s", f.Reason())
		}
		t.Errorf("SEC-003 findings = %d, want 4 (2 JS + 2 Py)", shellN)
	}
	if deserN != 2 { // pickle.loads + yaml.load
		for _, f := range findingsBy(res, "VX-SEC-013") {
			t.Logf("%s", f.Reason())
		}
		t.Errorf("SEC-013 findings = %d, want 2", deserN)
	}
	for _, f := range findingsBy(res, "VX-SEC-013") {
		if f.Severity != SevCritical || f.File != "server.py" {
			t.Errorf("deser finding wrong: %+v", f)
		}
	}
}

func TestPythonCleanFixture(t *testing.T) {
	res, err := Scan(fixtureDir("scanclean"), "c")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"VX-SEC-004", "VX-SEC-003", "VX-SEC-013"} {
		for _, f := range findingsBy(res, id) {
			if f.File == "safe.py" {
				t.Errorf("safe.py must not fire %s: %+v", id, f)
			}
		}
	}
}

func TestJSEmptyCatch(t *testing.T) {
	res, _ := Scan(fixtureDir("scanvuln"), "v")
	got := findingsBy(res, "VX-QUA-103")
	if len(got) != 1 {
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 1 empty-catch finding, got %d", len(got))
	}
	if got[0].Severity != SevLow || got[0].File != "swallow.js" {
		t.Errorf("empty catch finding wrong: %+v", got[0])
	}
}

func TestJSPIIScanners(t *testing.T) {
	res, _ := Scan(fixtureDir("scanvuln"), "v")
	got := findingsBy(res, "VX-PRV-302")
	if len(got) != 2 { // EMAIL_RE line + PHONE_RE line
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 PII-pattern findings, got %d", len(got))
	}
	for _, f := range got {
		if f.File != "pii.js" || f.Severity != SevMedium {
			t.Errorf("PII finding wrong: %+v", f)
		}
	}
}
