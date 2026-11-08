package scanner

import "regexp"

func init() {
	sourceRules = append(sourceRules, prvSecretInLogs)
}

// logCallRe: logging sinks whose arguments embed secrets.
var logCallRe = regexp.MustCompile(`\b(console|log|logger|logging)\s*[.(]\s*(log|error|warn|info|debug|trace)?\s*\(`)

// secretInArgRe: secret-ish expressions inside a logged argument — env keys
// must themselves look secret-bearing (PORT/HOST etc. are configuration).
var secretInArgRe = regexp.MustCompile(
	`process\.env\.[A-Za-z_]*(?:KEY|TOKEN|SECRET|PASSWORD|CRED)[A-Za-z_]*` +
		`|process\.env\[|authorization|api[_-]?key|token|secret|password`)

// argsUpToClose returns the full argument list text (up to the matching
// close paren, bounded) — secrets often sit in the 2nd+ argument.
func argsUpToClose(content string, after int) string {
	depth := 0
	end := after
	for end < len(content) && end-after < 300 {
		c := content[end]
		if c == '(' {
			depth++
		} else if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		}
		end++
	}
	return content[after:end]
}

// prvSecretInLogs (VX-PRV-303): logging statements that embed credentials —
// they end up in stderr captures, CI logs and crash reports.
var prvSecretInLogs = sourceRule{
	ID:    "VX-PRV-303",
	Title: "credentials written to logs",
	Sev:   SevMedium,
	Lang:  "js",
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range logCallRe.FindAllStringIndex(content, -1) {
			args := argsUpToClose(content, loc[1])
			if args == "" || !secretInArgRe.MatchString(args) {
				continue
			}
			line := lineOf(content, loc[0])
			snip := snippetAt(content, loc[0])
			if isCommentLine(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-PRV-303", Title: "credentials written to logs",
				Severity: SevMedium, File: rel, Line: line,
				Snippet: maskSnippet(snip),
				Why:     "secrets in log lines survive into CI logs and crash reports",
			})
		}
		return out
	},
}
