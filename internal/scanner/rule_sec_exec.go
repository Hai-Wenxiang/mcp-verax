package scanner

import (
	"regexp"
	"strings"
)

func init() {
	sourceRules = append(sourceRules, secEnvDump, secShellExec)
}

// envRefRe matches any process.env reference; whether it is a bare
// (whole-environment) reference is decided by the following rune — RE2 has
// no lookaheads. `process.env.HOME` / `process.env["X"]` are normal; a bare
// `process.env` (incl. spreads and call arguments) hands the entire
// environment, every secret in it, to the consumer.
var envRefRe = regexp.MustCompile(`process\.env\b`)

func bareEnvRef(content string, end int) bool {
	rest := content[end:]
	rest = strings.TrimLeft(rest, " \t")
	if rest == "" {
		return true // end of file: definitely bare
	}
	switch rest[0] {
	case '.', '[': // property access — normal use
		return false
	}
	return true
}

// secEnvDump (VX-SEC-001): exposing the whole environment — through tool
// results, logs or spreads — leaks every secret the server can see.
var secEnvDump = sourceRule{
	ID:    "VX-SEC-001",
	Title: "whole-environment exposure (bare process.env)",
	Sev:   SevCritical,
	Check: func(rel, content string) []Finding {
		var out []Finding
		for _, loc := range envRefRe.FindAllStringIndex(content, -1) {
			if !bareEnvRef(content, loc[1]) {
				continue
			}
			snip := snippetAt(content, loc[0])
			if isCommentLine(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-SEC-001", Title: "whole-environment exposure (bare process.env)",
				Severity: SevCritical, File: rel, Line: lineOf(content, loc[0]),
				Snippet: snip,
				Why:     "every env var (incl. secrets) becomes readable by the tool consumer",
			})
		}
		return out
	},
}

// shellTrueRe matches explicit shell:true options.
var shellTrueRe = regexp.MustCompile(`shell\s*:\s*true`)

// dynamicExecRe matches exec-family calls whose first argument does not
// start with a quote (i.e. a built/interpolated command string).
var dynamicExecRe = regexp.MustCompile(`\b(exec|execSync|spawn|spawnSync)\(\s*[^"'\s)]`)

// secShellExec (VX-SEC-003): shell command construction from data — string
// building into exec-family calls or explicit shell:true — is the classic
// command-injection surface.
var secShellExec = sourceRule{
	ID:    "VX-SEC-003",
	Title: "shell command built from non-literal input",
	Sev:   SevHigh,
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		flag := func(idx int, why string) {
			line := lineOf(content, idx)
			if seen[line] {
				return
			}
			snip := snippetAt(content, idx)
			if isCommentLine(snip) {
				return
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-003", Title: "shell command built from non-literal input",
				Severity: SevHigh, File: rel, Line: line, Snippet: snip, Why: why,
			})
		}
		for _, loc := range dynamicExecRe.FindAllStringIndex(content, -1) {
			flag(loc[0], "command string is constructed, not a literal — injection surface")
		}
		for _, loc := range shellTrueRe.FindAllStringIndex(content, -1) {
			flag(loc[0], "shell:true routes the command through a shell interpreter")
		}
		return out
	},
}
