package scanner

import (
	"regexp"
	"strings"
)

func init() {
	sourceRules = append(sourceRules, secEvalInput, secHardcodedSecrets)
}

// isCommentLine reports whether a trimmed source line is a JS/TS comment.
func isCommentLine(snippet string) bool {
	return strings.HasPrefix(snippet, "//") || strings.HasPrefix(snippet, "/*") ||
		strings.HasPrefix(snippet, "*") || strings.HasPrefix(snippet, "*/")
}

var evalRe = regexp.MustCompile(`\beval\s*\(|\bnew\s+Function\s*\(`)

// secEvalInput (VX-SEC-004): eval / new Function on server code is a
// remote-code-execution surface even before considering tool inputs.
var secEvalInput = sourceRule{
	ID:    "VX-SEC-004",
	Lang:  "js",
	Title: "dynamic code evaluation (eval / new Function)",
	Sev:   SevHigh,
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range evalRe.FindAllStringIndex(content, -1) {
			snip := snippetAt(content, loc[0])
			if isCommentLine(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-SEC-004", Title: "dynamic code evaluation (eval / new Function)",
				Severity: SevHigh, File: rel, Line: lineOf(content, loc[0]),
				Snippet: snip,
				Why:     "eval-family calls turn any injected string into executed code",
			})
		}
		return out
	},
}

// secretRes are credential SHAPES (prefix + charset + length), not real values.
var secretRes = []*regexp.Regexp{
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{16,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`ghp_[A-Za-z0-9]{30,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{10,}`),
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{30,}`),
}

// secHardcodedSecrets (VX-SEC-005): credential-shaped literals in source.
var secHardcodedSecrets = sourceRule{
	ID:    "VX-SEC-005",
	Title: "credential-shaped literal in source",
	Sev:   SevHigh,
	Check: func(rel, content string) []Finding {
		var out []Finding
		seenLines := map[int]bool{}
		for _, re := range secretRes {
			for _, loc := range re.FindAllStringIndex(content, -1) {
				line := lineOf(content, loc[0])
				if seenLines[line] {
					continue // one finding per line even if multiple shapes match
				}
				snip := snippetAt(content, loc[0])
				if isCommentLine(snip) {
					continue
				}
				seenLines[line] = true
				out = append(out, Finding{
					RuleID: "VX-SEC-005", Title: "credential-shaped literal in source",
					Severity: SevHigh, File: rel, Line: line,
					Snippet: maskSnippet(snip),
					Why:     "leaks at publish time; rotate the credential and load it from env",
				})
			}
		}
		return out
	},
}

// maskSnippet shortens anything that looks like a found credential.
func maskSnippet(snip string) string {
	if len(snip) > 60 {
		return snip[:60] + "…(masked)"
	}
	return snip
}
