package scanner

import "testing"

func TestRuleTelemetry(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-PRV-301")
	if len(got) == 0 {
		t.Fatal("segment.io telemetry must be flagged")
	}
	if got[0].File != "telemetry.js" || got[0].Severity != SevMedium {
		t.Errorf("telemetry finding wrong: %+v", got[0])
	}
	if res2, _ := Scan(fixtureDir("scanclean"), "c"); len(findingsBy(res2, "VX-PRV-301")) != 0 {
		t.Error("clean fixture must stay telemetry-free")
	}
}

func TestRuleFSWrite(t *testing.T) {
	vuln, _ := Scan(fixtureDir("scanvuln"), "v")
	got := findingsBy(vuln, "VX-SEC-002")
	if len(got) != 1 { // the concatenation; literal appendFile is a negative
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 1 uncontained-write finding, got %d", len(got))
	}

	clean, _ := Scan(fixtureDir("scanclean"), "c")
	if n := len(findingsBy(clean, "VX-SEC-002")); n != 0 {
		t.Fatalf("contained writes must not fire, got %d", n)
	}
}

func TestRuleUnrestrictedOutbound(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-SEC-008")
	if len(got) != 2 { // fetch(TRACK_HOST) + fetch(url); literal docs fetch excluded
		for _, f := range got {
			t.Logf("%s", f.Reason())
		}
		t.Fatalf("want 2 non-literal outbound findings, got %d", len(got))
	}
	for _, f := range got {
		if f.Severity != SevMedium || f.File != "telemetry.js" {
			t.Errorf("outbound finding wrong: %+v", f)
		}
	}
}
