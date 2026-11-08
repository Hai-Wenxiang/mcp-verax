package scanner

import (
	"regexp"
	"strings"
)

func init() {
	sourceRules = append(sourceRules, prvTelemetry, secFSWrite, secUnrestrictedOutbound)
}

// telemetryRes: known analytics/telemetry endpoints phoning home from a
// server that users never opted into.
var telemetryRes = []*regexp.Regexp{
	regexp.MustCompile(`segment\.(?:io|com)`),
	regexp.MustCompile(`mixpanel\.(?:io|com)`),
	regexp.MustCompile(`posthog\.(?:in|com)`),
	regexp.MustCompile(`amplitude\.(?:com|io)`),
	regexp.MustCompile(`google-analytics\.com`),
	regexp.MustCompile(`sentry\.(?:io|com)`),
	regexp.MustCompile(`statsig\.(?:com|io)`),
	regexp.MustCompile(`datadoghq\.(?:com|eu)`),
	regexp.MustCompile(`track\.(?:telemetry|metrics)\w*\.`),
}

// prvTelemetry (VX-PRV-301): embedded telemetry endpoints — undisclosed
// data collection from a tool server.
var prvTelemetry = sourceRule{
	ID:    "VX-PRV-301",
	Title: "embedded telemetry endpoint",
	Sev:   SevMedium,
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[string]bool{}
		for _, re := range telemetryRes {
			for _, loc := range re.FindAllStringIndex(content, -1) {
				snip := snippetAt(content, loc[0])
				if isCommentLine(snip) {
					continue
				}
				if seen[snip] {
					continue
				}
				seen[snip] = true
				out = append(out, Finding{
					RuleID: "VX-PRV-301", Title: "embedded telemetry endpoint",
					Severity: SevMedium, File: rel, Line: lineOf(content, loc[0]),
					Snippet: maskSnippet(snip),
					Why:     "usage data leaves the machine without the operator opting in",
				})
			}
		}
		return out
	},
}

// fsCallRe: filesystem write APIs (call site; argument inspected in code —
// RE2 has no lookaheads and quote-starting concatenations exist).
var fsCallRe = regexp.MustCompile(`\b(writeFile|writeFileSync|appendFile|appendFileSync|createWriteStream)\s*\(`)

// firstArg returns the raw text of the first call argument (up to a comma or
// closing paren at nesting depth 0, capped for sanity).
func firstArg(content string, after int) string {
	depth := 0
	end := after
	for end < len(content) && end-after < 200 {
		c := content[end]
		if c == '(' {
			depth++
		} else if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		} else if c == ',' && depth == 0 {
			break
		}
		end++
	}
	return content[after:end]
}

// nonLiteralPathArg: starts with a quote AND has no top-level '+' means a
// pure string literal; anything else (variables, concatenations) is built.
func nonLiteralPathArg(arg string) bool {
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	if a[0] != '"' && a[0] != '\'' && a[0] != '`' {
		return true // variable
	}
	return strings.Contains(a, "+") // "prefix" + var concatenation
}

// containmentRe: markers that the code restricts writes to a root.
var containmentRe = regexp.MustCompile(`(?i)(allowed|root|base|sandbox)[A-Za-z]*(dir|path)|startsWith|path\.resolve\(\s*root|relative\(`)

// secFSWrite (VX-SEC-002): filesystem writes to computed paths without any
// visible containment check in the same file.
var secFSWrite = sourceRule{
	ID:    "VX-SEC-002",
	Title: "filesystem write to non-literal path",
	Sev:   SevHigh,
	Check: func(rel, content string) []Finding {
		hits := fsCallRe.FindAllStringIndex(content, -1)
		if len(hits) == 0 {
			return nil
		}
		if containmentRe.MatchString(content) {
			// the file shows some form of root/containment discipline; stay quiet
			return nil
		}
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range hits {
			arg := firstArg(content, loc[1])
			if !nonLiteralPathArg(arg) {
				continue // literal destination: not an injection surface
			}
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
				RuleID: "VX-SEC-002", Title: "filesystem write to non-literal path",
				Severity: SevHigh, File: rel, Line: line, Snippet: snip,
				Why: "no containment check in this file — a crafted path can escape the workspace",
			})
		}
		return out
	},
}

// outboundRe: well-known network entry points (bare get( is deliberately
// excluded — Map.get et al. would drown the signal).
var outboundRe = regexp.MustCompile(`\bfetch\s*\(|\baxios\s*\.\s*(?:get|post|put)\s*\(|\bhttps?\s*\.\s*(?:get|request)\s*\(|\bgot\s*\(|\brequest\s*\(`)

// nonLiteralArg reports whether the first argument after the call opening
// starts with something other than a quote (RE2 has no lookaheads).
func nonLiteralArg(content string, end int) bool {
	rest := content[end:]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t' || rest[0] == '\n') {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		return false
	}
	switch rest[0] {
	case '"', '\'', '`':
		return false // literal URL
	}
	return true
}

// allowlistRe: markers of deliberate destination control.
var allowlistRe = regexp.MustCompile(`(?i)allowlist|allow_?list|whitelist|white_?list|allowedHosts|ALLOWED_HOSTS|permitted`)

// secUnrestrictedOutbound (VX-SEC-008): outbound HTTP to computed URLs with
// no visible allowlist — SSRF surface when tool inputs reach the URL.
var secUnrestrictedOutbound = sourceRule{
	ID:    "VX-SEC-008",
	Title: "outbound request to non-literal URL without allowlist",
	Sev:   SevMedium,
	Check: func(rel, content string) []Finding {
		if allowlistRe.MatchString(content) {
			return nil
		}
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range outboundRe.FindAllStringIndex(content, -1) {
			if !nonLiteralArg(content, loc[1]) {
				continue // literal destination: fine at this rule's precision
			}
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
				RuleID: "VX-SEC-008", Title: "outbound request to non-literal URL without allowlist",
				Severity: SevMedium, File: rel, Line: line, Snippet: snip,
				Why: "tool-controlled URLs can target internal services (SSRF) unless destinations are constrained",
			})
		}
		return out
	},
}
