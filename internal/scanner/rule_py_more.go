package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	sourceRules = append(sourceRules, pyEvalExec, pyShellExec, pyUnsafeDeserialize, jsEmptyCatch, jsPIIScanners)
}

// langOf returns "py" for Python sources, "js" for JS/TS, "" otherwise.
func langOf(rel string) string {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".py":
		return "py"
	case ".js", ".cjs", ".mjs", ".ts", ".cts", ".mts":
		return "js"
	}
	return ""
}

// pyEvalRe covers both eval() and exec() plus compile-to-code helpers.
var pyEvalRe = regexp.MustCompile(`\b(eval|exec)\s*\(|\b__import__\s*\(\s*['"]exec`)

// pyEvalExec (VX-SEC-004): Python dynamic code execution.
var pyEvalExec = sourceRule{
	ID: "VX-SEC-004", Title: "dynamic code execution (eval/exec)", Sev: SevHigh,
	Lang: "py",
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range pyEvalRe.FindAllStringIndex(content, -1) {
			snip := snippetAt(content, loc[0])
			if pyComment(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-SEC-004", Title: "dynamic code execution (eval/exec)",
				Severity: SevHigh, File: rel, Line: lineOf(content, loc[0]), Snippet: snip,
				Why: "exec-family calls execute whatever string reaches them",
			})
		}
		return out
	},
}

// pyShellRe: os.system / os.popen / shell=True subprocess.
var pyShellRe = regexp.MustCompile(`\bos\.(?:system|popen)\s*\(|shell\s*=\s*True`)

var pyShellExec = sourceRule{
	ID: "VX-SEC-003", Title: "shell command via interpreter", Sev: SevHigh,
	Lang: "py",
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range pyShellRe.FindAllStringIndex(content, -1) {
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if pyComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-003", Title: "shell command via interpreter",
				Severity: SevHigh, File: rel, Line: line, Snippet: snip,
				Why: "interpreter-routed commands turn crafted strings into execution",
			})
		}
		return out
	},
}

// pyUnsafeDeserRe: unsafe deserialization surfaces.
var pyUnsafeDeserRe = regexp.MustCompile(`\bpickle\.loads?\s*\(|\byaml\.load\s*\(\s*[^,)]+(?:,\s*)?\)|\bmarshal\.loads?\s*\(`)

var pyUnsafeDeserialize = sourceRule{
	ID: "VX-SEC-013", Title: "unsafe deserialization (pickle/yaml.load/marshal)", Sev: SevCritical,
	Lang: "py",
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range pyUnsafeDeserRe.FindAllStringIndex(content, -1) {
			snip := snippetAt(content, loc[0])
			if pyComment(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-SEC-013", Title: "unsafe deserialization (pickle/yaml.load/marshal)",
				Severity: SevCritical, File: rel, Line: lineOf(content, loc[0]), Snippet: snip,
				Why: "deserializing untrusted bytes is arbitrary code execution by design",
			})
		}
		return out
	},
}

// pyComment: '#' comments.
func pyComment(snip string) bool { return strings.HasPrefix(snip, "#") }

// jsEmptyCatchRe: swallow-all catch blocks.
var jsEmptyCatchRe = regexp.MustCompile(`\bcatch\s*(?:\([^)]*\))?\s*\{\s*\}`)

var jsEmptyCatch = sourceRule{
	ID: "VX-QUA-103", Title: "empty catch block (swallowed error)", Sev: SevLow,
	Lang: "js",
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range jsEmptyCatchRe.FindAllStringIndex(content, -1) {
			out = append(out, Finding{
				RuleID: "VX-QUA-103", Title: "empty catch block (swallowed error)",
				Severity: SevLow, File: rel, Line: lineOf(content, loc[0]),
				Snippet: snippetAt(content, loc[0]),
				Why:     "failures vanish silently — tools report success while broken",
			})
		}
		return out
	},
}

// piiScannerRes: signatures of hardcoded personal-data classifiers — these
// match the *pattern source* (regex literals like \d{3}-\d{3}-\d{4} or the
// %+- email char-class), not actual PII values.
var piiScannerRes = []*regexp.Regexp{
	regexp.MustCompile(`\\d\{3\}[^"\n]{0,12}\\d\{(?:2|4)\}`), // phone/SSN pattern source
	regexp.MustCompile(`%\+[-*]`),                            // email char-class tail
}

var jsPIIScanners = sourceRule{
	ID: "VX-PRV-302", Title: "hardcoded PII-matching patterns (emails/phones/SSN)", Sev: SevMedium,
	Lang: "js",
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		for _, re := range piiScannerRes {
			for _, loc := range re.FindAllStringIndex(content, -1) {
				line := lineOf(content, loc[0])
				if seen[line] {
					continue
				}
				snip := snippetAt(content, loc[0])
				if isCommentLine(snip) {
					continue
				}
				seen[line] = true
				out = append(out, Finding{
					RuleID: "VX-PRV-302", Title: "hardcoded PII-matching patterns (emails/phones/SSN)",
					Severity: SevMedium, File: rel, Line: line, Snippet: maskSnippet(snip),
					Why: "the server classifies personal data it processes — disclose this to operators",
				})
			}
		}
		return out
	},
}
