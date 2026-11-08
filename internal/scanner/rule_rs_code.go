package scanner

import (
	"regexp"
	"strings"
)

func init() {
	sourceRules = append(sourceRules,
		rsCommandExec, rsFSWrite, rsOutbound, rsUnsafe, rsPanicUnwrap, rsSecretInLogs)
}

// rsComment: '//' line comments.
func rsComment(snip string) bool { return strings.HasPrefix(snip, "//") }

// rsCommandNewRe: process spawning entry point.
var rsCommandNewRe = regexp.MustCompile(`\bCommand\s*::\s*new\s*\(`)

// rsShells: interpreter programs that turn strings into scripts.
var rsShells = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true,
	"cmd": true, `"cmd`: true, "/bin/sh": true, "/bin/bash": true,
}

// rsCommandExec (VX-SEC-003/rs): process spawning with a computed program
// or an interpreter shell — the Rust equivalent of string-built commands.
var rsCommandExec = sourceRule{
	ID: "VX-SEC-003", Title: "process spawn with dynamic program or shell", Sev: SevHigh,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range rsCommandNewRe.FindAllStringIndex(content, -1) {
			arg := strings.TrimSpace(firstArg(content, loc[1]))
			if arg == "" {
				continue
			}
			if arg[0] == '"' {
				prog := strings.Trim(arg, `"`)
				if !rsShells[prog] {
					continue // fixed literal binary: arg-vector spawn, no string shell
				}
			} else if arg[0] == 'r' && len(arg) > 1 && arg[1] == '#' {
				continue // raw string literal program: fixed
			}
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-003", Title: "process spawn with dynamic program or shell",
				Severity: SevHigh, File: rel, Line: line, Snippet: maskSnippet(snip),
				Why: "computed program or interpreter shell turns crafted strings into execution",
			})
		}
		return out
	},
}

// rsFSWriteRe: filesystem write entry points.
var rsFSWriteRe = regexp.MustCompile(`\bfs::write\s*\(|\bFile::create\s*\(`)

// rsContainmentRe: markers of write-root discipline (Rust flavor).
var rsContainmentRe = regexp.MustCompile(`(?i)starts_with|canonicalize|allowed|root_?dir|base_?dir|sandbox|sanitize`)

// rsNonLiteralPath: Rust string literals start with '"' or 'r"'; anything
// else (variables, format!) is a computed path.
func rsNonLiteralPath(arg string) bool {
	a := strings.TrimSpace(arg)
	if a == "" {
		return false
	}
	if a[0] == '"' {
		return false
	}
	if a[0] == 'r' && len(a) > 1 && a[1] == '#' {
		return false
	}
	return true
}

// rsFSWrite (VX-SEC-002/rs): filesystem writes to computed paths without
// visible containment in the same file.
var rsFSWrite = sourceRule{
	ID: "VX-SEC-002", Title: "filesystem write to non-literal path", Sev: SevHigh,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		hits := rsFSWriteRe.FindAllStringIndex(content, -1)
		if len(hits) == 0 || rsContainmentRe.MatchString(content) {
			return nil
		}
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range hits {
			if !rsNonLiteralPath(firstArg(content, loc[1])) {
				continue
			}
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-002", Title: "filesystem write to non-literal path",
				Severity: SevHigh, File: rel, Line: line, Snippet: maskSnippet(snip),
				Why: "no containment check in this file — a crafted path can escape the workspace",
			})
		}
		return out
	},
}

// rsHTTPClientRe: evidence the file talks HTTP (gates the outbound rule —
// bare .get( is HashMap noise in Rust too).
var rsHTTPClientRe = regexp.MustCompile(`reqwest|ureq|hyper::|http::`)

// rsOutboundCallRe: request builder calls.
var rsOutboundCallRe = regexp.MustCompile(`\.(?:get|post|put|delete|request)\s*\(`)

// rsOutbound (VX-SEC-008/rs): outbound HTTP to computed URLs with no
// visible allowlist — SSRF surface when tool inputs reach the URL.
var rsOutbound = sourceRule{
	ID: "VX-SEC-008", Title: "outbound request to non-literal URL without allowlist", Sev: SevMedium,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		if !rsHTTPClientRe.MatchString(content) || allowlistRe.MatchString(content) {
			return nil
		}
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range rsOutboundCallRe.FindAllStringIndex(content, -1) {
			if !nonLiteralArg(content, loc[1]) {
				continue // literal URL: fine at this rule's precision
			}
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-008", Title: "outbound request to non-literal URL without allowlist",
				Severity: SevMedium, File: rel, Line: line, Snippet: maskSnippet(snip),
				Why: "tool input reaching the URL makes the server an SSRF pivot",
			})
		}
		return out
	},
}

// rsUnsafeRe: unsafe code introductions.
var rsUnsafeRe = regexp.MustCompile(`\bunsafe\s+(?:fn|impl|trait)\b|\bunsafe\s*\{`)

// rsUnsafe (VX-SEC-014): unsafe Rust — raw pointers, FFI, manual memory.
var rsUnsafe = sourceRule{
	ID: "VX-SEC-014", Title: "unsafe code block or impl", Sev: SevMedium,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range rsUnsafeRe.FindAllStringIndex(content, -1) {
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-SEC-014", Title: "unsafe code block or impl",
				Severity: SevMedium, File: rel, Line: line, Snippet: maskSnippet(snip),
				Why: "unsafe bypasses Rust's guarantees — each block is a manual audit item",
			})
		}
		return out
	},
}

// rsPanicRe: panic surfaces on request paths.
var rsPanicRe = regexp.MustCompile(`\.unwrap\s*\(\s*\)|\.expect\s*\(`)

// rsSkipDirs: paths where panicking is conventional.
func rsSkipDirs(rel string) bool {
	return strings.HasPrefix(rel, "tests/") || strings.HasPrefix(rel, "benches/") ||
		strings.HasPrefix(rel, "examples/") || strings.HasPrefix(rel, "testfixtures/")
}

// rsPanicUnwrap (VX-QUA-107): unwrap/expect panics — a malformed tool input
// crashes the whole server (availability, not just correctness).
var rsPanicUnwrap = sourceRule{
	ID: "VX-QUA-107", Title: "unwrap/expect panic surface", Sev: SevLow,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		if rsSkipDirs(rel) {
			return nil
		}
		var out []Finding
		for _, loc := range rsPanicRe.FindAllStringIndex(content, -1) {
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			out = append(out, Finding{
				RuleID: "VX-QUA-107", Title: "unwrap/expect panic surface",
				Severity: SevLow, File: rel, Line: lineOf(content, loc[0]), Snippet: maskSnippet(snip),
				Why: "panics on request paths take the server down instead of erroring the call",
			})
		}
		return out
	},
}

// rsLogCallRe: Rust logging sinks.
var rsLogCallRe = regexp.MustCompile(`\b(?:println|eprintln|print|eprint)!\s*\(|\blog\s*::\s*(?:info|debug|trace|warn)\s*\(`)

// rsSecretInArgRe: secret-bearing expressions in Rust sources.
var rsSecretInArgRe = regexp.MustCompile(
	`env::var\s*\(\s*"[^"]*(?:KEY|TOKEN|SECRET|PASSWORD|CRED)[^"]*"` +
		`|authorization|api[_-]?key|token|secret|password`)

// rsSecretInLogs (VX-PRV-303/rs): logging statements that embed credentials.
var rsSecretInLogs = sourceRule{
	ID: "VX-PRV-303", Title: "credentials written to logs", Sev: SevMedium,
	Lang: "rs",
	Check: func(rel, content string) []Finding {
		var out []Finding
		seen := map[int]bool{}
		for _, loc := range rsLogCallRe.FindAllStringIndex(content, -1) {
			args := argsUpToClose(content, loc[1])
			if !rsSecretInArgRe.MatchString(args) {
				continue
			}
			line := lineOf(content, loc[0])
			if seen[line] {
				continue
			}
			snip := snippetAt(content, loc[0])
			if rsComment(snip) {
				continue
			}
			seen[line] = true
			out = append(out, Finding{
				RuleID: "VX-PRV-303", Title: "credentials written to logs",
				Severity: SevMedium, File: rel, Line: line, Snippet: maskSnippet(snip),
				Why: "logged credentials end up in stderr captures, CI logs and crash reports",
			})
		}
		return out
	},
}
