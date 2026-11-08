package scanner

import "testing"

func TestRuleSecretInLogs(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "v")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-PRV-303")
	if len(got) != 2 {
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 log-leak findings, got %d", len(got))
	}
	for _, f := range got {
		if f.File != "logleak.js" || f.Severity != SevMedium {
			t.Errorf("log leak finding wrong: %+v", f)
		}
	}

	clean, _ := Scan(fixtureDir("scanclean"), "c")
	if n := len(findingsBy(clean, "VX-PRV-303")); n != 0 {
		for _, f := range findingsBy(clean, "VX-PRV-303") {
			t.Logf("%s", f.Reason())
		}
		t.Errorf("clean fixture must stay clean, got %d", n)
	}
}
