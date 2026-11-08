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

// envInheritanceContext: spreading/copying env into a child-process spawn
// is the standard inheritance pattern, not consumer-facing exposure. The
// spawn call and the env spread are often a few lines apart (options
// object), so the previous few lines count as context.
var envInheritanceRe = regexp.MustCompile(`\b(spawn|spawnSync|execFile|fork|exec)\s*\(`)

// envDefaultParam matches the DI idiom `func(env = process.env, ...)` —
// `process.environment` et al. excluded via the negated word char.
var envDefaultParamRe = regexp.MustCompile(`=\s*process\.env[^a-zA-Z0-9_]`)

func envDefaultParam(snip string) bool { return envDefaultParamRe.MatchString(snip) }

func envInheritanceContext(content string, at int, snip string) bool {
	if envInheritanceRe.MatchString(snip) {
		return true
	}
	// look back up to 3 line starts for the spawn call
	starts := 0
	i := at
	for i > 0 && starts < 3 {
		if content[i-1] == '\n' {
			starts++
		}
		i--
	}
	return envInheritanceRe.MatchString(content[i:at])
}

// secEnvDump (VX-SEC-001): exposing the whole environment — through tool
// results, logs or spreads — leaks every secret the server can see.
var secEnvDump = sourceRule{
	ID:    "VX-SEC-001",
	Lang:  "js",
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
			if envInheritanceContext(content, loc[0], snip) {
				continue // ...process.env into a child spawn: inheritance, not exposure
			}
			if envDefaultParam(snip) {
				out = append(out, Finding{
					RuleID: "VX-SEC-001", Title: "env dependency-injection default (review outputs)",
					Severity: SevMedium, File: rel, Line: lineOf(content, loc[0]),
					Snippet: snip,
					Why:     "process.env is the default argument — fine for testing, but verify the function never returns env contents to tool callers",
				})
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

// spawnWithArgArray reports whether the match is a spawn/spawnSync call
// whose second argument is an array literal (argv form — safe by design).
func spawnWithArgArray(content string, loc []int) bool {
	m := content[loc[0]:loc[1]]
	if !strings.Contains(m, "spawn") {
		return false
	}
	rest := content[loc[1]:]
	if len(rest) > 60 {
		rest = rest[:60]
	}
	rest = strings.TrimLeft(rest, " 	")
	if strings.HasPrefix(rest, "(") { // second arg group started; skip first arg
		rest = rest[1:]
	}
	// find the comma that ends the first argument at depth 0
	depth := 0
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return false
			}
			depth--
		case ',':
			if depth == 0 {
				nxt := strings.TrimLeft(rest[i+1:], " \t\n")
				return strings.HasPrefix(nxt, "[")
			}
		}
	}
	return false
}

// secShellExec (VX-SEC-003): shell command construction from data — string
// building into exec-family calls or explicit shell:true — is the classic
// command-injection surface.
var secShellExec = sourceRule{
	ID:    "VX-SEC-003",
	Lang:  "js",
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
			if spawnWithArgArray(content, loc) {
				continue // argv-array spawn: the recommended safe form
			}
			flag(loc[0], "command string is constructed, not a literal — injection surface")
		}
		for _, loc := range shellTrueRe.FindAllStringIndex(content, -1) {
			flag(loc[0], "shell:true routes the command through a shell interpreter")
		}
		return out
	},
}
