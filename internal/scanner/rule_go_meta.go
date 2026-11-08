package scanner

import (
	"strings"
)

func init() {
	packageRules = append(packageRules, goMeta)
}

// goMeta (VX-QUA-105/go): go.mod hygiene — module path present, go directive
// present; repository provenance comes from the LICENSE/README files which
// the JS rule already checks via its own paths (README) — here we check LICENSE.
var goMeta = packageRule{
	ID:    "VX-QUA-105",
	Title: "go.mod hygiene",
	Sev:   SevLow,
	Check: func(rel string, data []byte) []Finding {
		if !strings.HasSuffix(rel, "go.mod") || len(data) == 0 {
			return nil
		}
		text := string(data)
		var out []Finding
		if !strings.Contains(text, "\nmodule ") && !strings.HasPrefix(text, "module ") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "go.mod has no module directive",
				Severity: SevMedium, File: rel,
				Why: "not a valid module; provenance unknown",
			})
		}
		if !strings.Contains(text, "\ngo ") && !strings.HasPrefix(text, "go ") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "go.mod has no go directive",
				Severity: SevLow, File: rel,
				Why: "language version undeclared; builds drift across toolchains",
			})
		}
		return out
	},
}
