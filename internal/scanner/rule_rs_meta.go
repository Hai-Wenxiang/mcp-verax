package scanner

import "strings"

func init() {
	packageRules = append(packageRules, rsCargoMeta)
}

// rsCargoMeta (VX-QUA-105/rs): Cargo.toml [package] hygiene — license,
// repository and description are the minimum provenance signals.
var rsCargoMeta = packageRule{
	ID:    "VX-QUA-105",
	Title: "Cargo.toml hygiene",
	Sev:   SevLow,
	Check: func(rel string, data []byte) []Finding {
		if !strings.HasSuffix(rel, "Cargo.toml") || len(data) == 0 {
			return nil
		}
		text := string(data)
		if !strings.Contains(text, "[package]") {
			return nil // workspace root manifest: members carry their own
		}
		var out []Finding
		if !strings.Contains(text, "license") && !strings.Contains(text, "license-file") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no license declared", Severity: SevLow,
				File: rel, Why: "users cannot legally rely on the crate",
			})
		}
		if !strings.Contains(text, "repository") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no repository link", Severity: SevLow,
				File: rel, Why: "no source provenance to audit against",
			})
		}
		if !strings.Contains(text, "description") {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no description", Severity: SevLow,
				File: rel, Why: "listing gives users no signal about intent",
			})
		}
		return out
	},
}
