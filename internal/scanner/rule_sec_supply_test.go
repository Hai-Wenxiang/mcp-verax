package scanner

import "testing"

func TestRuleInstallScripts(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-SEC-012")
	if len(got) != 1 {
		t.Fatalf("want 1 install-script finding, got %d: %+v", len(got), got)
	}
	if got[0].Severity != SevMedium || !contains(got[0].Snippet, "preinstall") {
		t.Errorf("finding wrong: %+v", got[0])
	}

	// benign script values must not fire
	if f := secInstallScripts.Check("p.json", []byte(`{"scripts":{"preinstall":"exit 0","postinstall":"echo built"}}`)); len(f) != 0 {
		t.Errorf("benign scripts flagged: %+v", f)
	}
}

func TestRuleTyposquat(t *testing.T) {
	res, err := Scan(fixtureDir("scanvuln"), "scanvuln-fixture")
	if err != nil {
		t.Fatal(err)
	}
	got := findingsBy(res, "VX-SEC-011")
	if len(got) != 1 {
		t.Fatalf("want 1 typosquat finding (lodahs≈lodash), got %d: %+v", len(got), got)
	}
	if got[0].Severity != SevMedium { // transposition = distance 2
		t.Errorf("lodahs should be medium: %+v", got[0])
	}

	// distance-1 hits are high
	if f := secTyposquat.Check("p.json", []byte(`{"dependencies":{"axio":"1.0.0"}}`)); len(f) != 1 || f[0].Severity != SevHigh {
		t.Errorf("distance-1 typo should be high: %+v", f)
	}

	// exact popular names are fine
	if f := secTyposquat.Check("p.json", []byte(`{"dependencies":{"lodash":"4.0.0","axios":"1.0.0"}}`)); len(f) != 0 {
		t.Errorf("exact popular names flagged: %+v", f)
	}
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"lodahs", "lodash", 2}, // transposition
		{"axios", "axios", 0},
		{"axio", "axios", 1},
		{"request", "reqeust", 2},
		{"abc", "xyz", 3},
	}
	for _, c := range cases {
		if got := editDistance(c.a, c.b); got != c.want {
			t.Errorf("editDistance(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
