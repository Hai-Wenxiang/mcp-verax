package scanner

import (
	"strings"
	"testing"
)

func TestAllRulesCatalog(t *testing.T) {
	all := AllRules()
	if len(all) != 16 {
		t.Fatalf("catalog = %d unique rules, want 16", len(all))
	}
	seen := map[string]bool{}
	for _, r := range all {
		if seen[r.ID] {
			t.Errorf("duplicate rule id: %s", r.ID)
		}
		seen[r.ID] = true
		switch r.ID {
		case "VX-SEC-001", "VX-SEC-005", "VX-SEC-006", "VX-QUA-105", "VX-PRV-303":
			if r.Title == "" || r.Severity == "" {
				t.Errorf("rule %s incomplete: %+v", r.ID, r)
			}
		case "VX-SEC-003", "VX-SEC-004":
			if !strings.Contains(r.Lang, "js") || !strings.Contains(r.Lang, "py") {
				t.Errorf("rule %s should merge languages, got %q", r.ID, r.Lang)
			}
		}
	}
}
