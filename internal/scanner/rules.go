package scanner

import "strings"

// sourceRule inspects one source file's full text.
type sourceRule struct {
	ID    string
	Title string
	Sev   Severity
	Check func(rel, content string) []Finding
}

// packageRule inspects package.json (nil data = file missing).
type packageRule struct {
	ID    string
	Title string
	Sev   Severity
	Check func(rel string, data []byte) []Finding
}

var packageRules []packageRule
var sourceRules []sourceRule

func init() {
	// Rules register themselves from their own files via init() append, so
	// this file never reassigns (and wipes) the registries.
	packageRules = append(packageRules, quaMetaMissing)
}

// lineOf returns the 1-based line number of the i-th byte.
func lineOf(content string, i int) int {
	return 1 + strings.Count(content[:i], "\n")
}

// snippetAt returns the trimmed source line containing index i.
func snippetAt(content string, i int) string {
	start := strings.LastIndexByte(content[:i], '\n') + 1
	end := strings.IndexByte(content[i:], '\n')
	if end < 0 {
		end = len(content)
	} else {
		end = i + end
	}
	return strings.TrimSpace(content[start:end])
}

// quaMetaMissing (VX-QUA-105): missing license / repository / description —
// the minimum metadata for a trustworthy package listing.
var quaMetaMissing = packageRule{
	ID:    "VX-QUA-105",
	Title: "package metadata incomplete",
	Sev:   SevLow,
	Check: func(rel string, data []byte) []Finding {
		if len(data) == 0 {
			return []Finding{{
				RuleID: "VX-QUA-105", Title: "package.json missing", Severity: SevMedium,
				File: rel, Why: "no package manifest at all: version, license and provenance unknown",
			}}
		}
		pkg, ok := parsePackageJSON(data)
		if !ok {
			return []Finding{{
				RuleID: "VX-QUA-105", Title: "package.json not valid JSON", Severity: SevMedium,
				File: rel, Why: "manifest cannot be parsed; provenance unknown",
			}}
		}
		var out []Finding
		if pkg.License == nil || pkg.License == "" {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no license declared", Severity: SevLow,
				File: rel, Why: "users cannot legally rely on the package",
			})
		}
		if pkg.Repository == nil {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no repository link", Severity: SevLow,
				File: rel, Why: "no source provenance to audit against",
			})
		}
		if pkg.Description == "" {
			out = append(out, Finding{
				RuleID: "VX-QUA-105", Title: "no description", Severity: SevLow,
				File: rel, Why: "listing gives users no signal about intent",
			})
		}
		return out
	},
}
