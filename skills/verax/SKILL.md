---
name: verax
description: >-
  Test, audit and score MCP servers with the Verax toolkit. Use when the user
  builds, publishes or reviews an MCP server and wants: protocol compliance
  tests from a declarative YAML suite, security/quality/privacy scanning with
  a 0-100 score, SARIF output, CI gates, fuzzing or flake detection. Trigger
  phrases include "test my MCP server", "scan this MCP server", "is this MCP
  server safe", "MCP server security audit", "MCP CI gate", "SARIF for the
  MCP server", "score this MCP server". Verax is a single Go binary — no
  runtime dependencies.
---

# Verax — test, audit and score MCP servers

Verax ("Registry discovers. Verax verifies.") is a single static Go binary
that (1) runs declarative YAML test suites against a live MCP server and
(2) statically scans MCP server source or its *published artifact* for
security/quality/privacy issues, producing a 0–100 score.

## Install (pin a version, verify the checksum)

Never curl-and-run binaries into an agent's shell without verification.
Install for the current platform and verify against the published
SHA256SUMS:

```bash
V=$(curl -fsSL https://api.github.com/repos/Hai-Wenxiang/mcp-verax/releases/latest | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)
curl -fsSL -o verax "https://github.com/Hai-Wenxiang/mcp-verax/releases/download/${V}/verax-$(uname -s | tr A-Z a-z)-$(uname -m | sed s/x86_64/amd64/;s/aarch64/arm64/)"
curl -fsSL -o SHA256SUMS "https://github.com/Hai-Wenxiang/mcp-verax/releases/download/${V}/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS   # must pass before chmod
chmod +x verax && ./verax version               # should print ${V}
```

Windows: download `verax-windows-amd64.exe` from the same release and
`Get-FileHash` against `SHA256SUMS`. Already have Go? `go install
github.com/Hai-Wenxiang/mcp-verax/cmd/verax@${V}` skips the manual check.

## Which command to use

| Situation | Command |
|---|---|
| User has a *running* MCP server and wants tests | `verax init tests/x.suite.yaml -- <server command>` — generates a green starter suite from the live server, then edit expectations |
| Run a suite (assertions, snapshots, latency SLO, process-leak, fuzz) | `verax test suite.yaml` · add `--repeat 10` for flake detection, `-j 4` for parallel suites, `--html report.html` / `--junit r.xml` for reports |
| Audit source or a *published package* | `verax scan <target>` where target is a directory, `@scope/pkg`, `pkg@version`, `crate:name[@version]`, `pypi:name[@version]` or `gh:owner/repo[@ref]` |
| CI gate | `verax scan <target> --min-score 90` or `--fail-level high` (exit 1 on findings at/above severity) |
| Scaffold a new server (suite-green, self-scan 100/100) | `verax new <template> <name>` — templates: filesystem-safe, csv-query, notes-store, rest-api-wrap, web-fetch-md, db-readonly |
| What rules exist | `verax rules` |

## Reading scan output

Findings print as `[SEVERITY] rule-id file:line: title` with a masked
snippet and a one-line "why". Severity weights: critical 25 / high 10 /
medium 4 / low 1; score = 100 − Σ weights, floored at 0. Machine-readable
outputs: `--sarif out.sarif` (GitHub Security tab), `--json out.json`,
`--badge b.svg`.

Interpretation guidance: findings in a scanned package are *evidence to
report*, not automatically a refusal — a low score on a third-party
package is information for the user, and fix guidance per rule lives in
the repo's `docs/rules.md`.

## Pitfalls

- `verax scan` exits 1 whenever findings exist (no gate flags = threshold
  low); in scripts, prefer explicit `--min-score`/`--fail-level`.
- `verax test` needs the server command after `--` in `init`; suites run
  the server themselves, so don't start it manually first.
- Scanning npm/crate/pypi/gh targets downloads the published artifact —
  expect a few seconds of network; use a local directory to stay offline.
