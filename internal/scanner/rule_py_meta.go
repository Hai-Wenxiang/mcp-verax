package scanner

import (
	"strings"
)

func init() {
	packageRules = append(packageRules, pyMeta)
}

// pyMeta (VX-QUA-105/py): pyproject.toml [project] metadata completeness —
// the Python twin of the package.json rule. Parsed with a pragmatic
// section-scanner: no TOML dependency, only the keys we judge.
var pyMeta = packageRule{
	ID:    "VX-QUA-105",
	Title: "pyproject metadata incomplete",
	Sev:   SevLow,
	Check: func(rel string, data []byte) []Finding {
		if !strings.HasSuffix(rel, "pyproject.toml") || len(data) == 0 {
			return nil
		}
		text := string(data)
		if !strings.Contains(text, "[project]") {
			// setuptools legacy layout: check setup.py presence indirectly via deps
			return []Finding{{
				RuleID: "VX-QUA-105", Title: "pyproject has no [project] table",
				Severity: SevMedium, File: rel,
				Why: "modern Python packaging metadata (PEP 621) missing; provenance unknown",
			}}
		}
		var out []Finding
		if !tomlHasKey(text, "license") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no license declared (pyproject)",
				Severity: SevLow, File: rel,
				Why: "users cannot legally rely on the package",
			})
		}
		if !tomlHasKey(text, "description") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no description (pyproject)",
				Severity: SevLow, File: rel,
				Why: "listing gives users no signal about intent",
			})
		}
		if !strings.Contains(text, "repository") && !strings.Contains(text, "urls") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no repository link (pyproject)",
				Severity: SevLow, File: rel,
				Why: "no source provenance to audit against",
			})
		}
		return out
	},
}

// tomlHasKey reports whether the [project] section declares key = at top level.
func tomlHasKey(text, key string) bool {
	inProject := false
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			inProject = t == "[project]"
			continue
		}
		if !inProject {
			continue
		}
		if strings.HasPrefix(t, key) {
			rest := strings.TrimPrefix(t, key)
			if strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "=") {
				return true
			}
		}
	}
	return false
}
