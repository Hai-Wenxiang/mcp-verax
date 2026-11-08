# Verax Status / 项目进展

> Overall progress of Verax components. Updated per feature release.
> Verax 各组件总体进展，随每个功能完成而更新。

## Current Milestone: M1 — Verax Test Kit / 当前里程碑：M1 Test Kit

| Feature | Status | Notes |
|---|---|---|
| F0 Repo bootstrap (Go module, CI, ADR-001, AGENTS.md) | ✅ Done | CI matrix gofmt/vet/test/build ×3 OS |
| F1 Protocol client (JSON-RPC, stdio transport, MCP handshake, primitives) | ✅ Done | 22 tests, 2 fixtures (echo/mcp) |
| F2 Suite DSL parser (YAML schema, validation, env expansion) | ✅ Done | 13 tests; strict unknown-field rejection |
| F3 Assertion engine (assertions + snapshots) | ✅ Done | 15 tests; jest-style golden snapshots |
| F4 Runner + CLI (`verax test`) + terminal reporter | ✅ Done | shared/respawn modes; example suite smoke 3/3 |
| F5 SLO (latency p95, concurrency) + process-leak + hang detection | ✅ Done | ns-precision timing; tree kill; leak(process_exit) case |
| F6 Schema-driven fuzzing + injection dictionaries | ✅ Done | seeded/deterministic; crash aborts campaign; inert dictionary |
| F7 Reporters (JUnit XML, static HTML) | ✅ Done | single-file HTML; CLI --junit/--html; flags-after-files fixed |
| F8 GitHub Action + release pipeline | ✅ Done | composite action + self-test workflow + tag→5-platform release |
| F9 Bootstrap E2E + first compatibility report | ✅ Done | 4 official servers driven; 2 Verax bugs found+fixed; F-2 server bug isolated (see compat/FINDINGS.md) |

## Current Milestone: M2 — Verax Scanner / 当前里程碑：M2 Scanner

| Feature | Status | Notes |
|---|---|---|
| S1 Engine core (findings/severity/score/walk) + QUA-105/106 | ✅ Done | 25/10/4/1 weights, floor 0 |
| S2 VX-SEC-004 eval + VX-SEC-005 credential literals | ✅ Done | 7 credential shapes, masked snippets |
| S3 VX-SEC-001 whole-env exposure + VX-SEC-003 shell construction | ✅ Done | negative-lookahead-free bare-env detection |
| S4 VX-SEC-012 install hooks + VX-SEC-011 typosquat | ✅ Done | edit distance 1(high)/2(medium) |
| S5 npm pack targets (audit the published artifact) | ✅ Done | array+object npm JSON; escape-proof tar |
| S6 SARIF 2.1.0 + `verax scan` CLI + e2e | ✅ Done | real scans: memory 96/100, everything 71/100 |
| S7 Privacy/outbound rules (PRV-301, SEC-002, SEC-008) | ✅ Done | e2e found everything's url-elicitation surface |
| S8 Action scan mode + self-test job | ✅ Done | mode: scan with sarif artifact upload |
| S9 Language-aware rule dispatch + Python pack + JS QUA/PRV | ✅ Done | 16 rules total (js+py+pkg) |
| deps CVE (OSV) + per-advisory severity | ✅ Done | S10/S11, best-effort refinement |
| **Registry Intelligence v0 (`verax registry-pulse`)** | ✅ Done | live official registry pull → sample scans → JSON+MD; nightly workflow; first live pulse: 2 npm pkgs (88, 95); DI-default false-positive class found & fixed via live data |
| badge service (Worker), M4 leaderboard | ⬜ Next | |

Status legend: ⬜ Planned ｜ 🚧 In progress ｜ ✅ Done ｜ ⏸ Deferred

## Later Versions / 后续版本

| Component | Milestone | Status |
|---|---|---|
| Verax Scanner (security/quality/privacy rule packs, SARIF, score) | M2 | ⬜ Not started |
| Verax Templates (TS + Python scaffolds) | M3 | ⬜ Not started |
| Verax Registry Intelligence (mirror, leaderboard, quarterly report) | M4 | ⬜ Not started |
| MCP gateway | — | ⏸ Evaluated, deferred (crowded space; see research docs) |

## Changelog / 变更日志

- 2026-11-08 — M1 development started. ADR-001 accepted: Test Kit in Go.
- 2026-11-08 — F0–F8 complete: protocol client, DSL, assertions+snapshots,
  runner/CLI, SLO, leak/hang, fuzz, JUnit/HTML reporters, Action+release.
- 2026-11-08 — **v0.1.0 released**: CI gate green (ci ×3OS + action-test ×2OS);
  5-platform binaries + SHA256SUMS published via release pipeline.
- 2026-11-08 — F9 compat bootstrap: 4 official servers, 2 Verax bug fixes
  (capability-gated discovery, handshake stderr), F-2/F-3 findings documented.
- 2026-11-08 — **M2 Scanner core (S1–S6)**: engine + 8 rules + SARIF +
  npm-artifact targets; real-package e2e on official servers.
- 2026-11-08 — M2 S7–S9: privacy/outbound rules, Action scan mode, Python
  pack + language-aware dispatch — **16 rules**, all gates green.
- 2026-11-08 — S10–S13: osv.dev CVE enrichment, log-secret rule, verax rules
  catalog + handbook, --min-score gate, Streamable HTTP url-target suites.
- 2026-11-08 — **v0.2.0 released**.
- 2026-11-08 — `verax new` scaffolder (templates embedded), notes-store
  template, VX-DYN-201 SSRF dynamic probe. **v0.3.0 released**.

## M3 kickoff: templates (2026-11-08 evening)

| Feature | Status | Notes |
|---|---|---|
| filesystem-safe template (first of the library) | ✅ Done | zero-dep Node, root containment; suite 5/5, self-scan 100/100 |
| csv-query template | ✅ Done | zero-dep, caps + containment; suite 7/7, self-scan 100/100 |
| notes-store template | ✅ Done | zero-dep CRUD, id sanitize + caps; suite 8/8, self-scan 100/100 |
| `verax new` scaffolder | ✅ Done | templates embedded in binary; scaffold→cd→suite e2e green |
| VX-DYN-201 SSRF probe (suite-level dynamic) | ✅ Done |
| VX-DYN-202 side-effect probe (honeypot diff) | ✅ Done | write-while-read-only detection, per-tool change evidence |
| Python project scanning (pyproject.toml) | ✅ Done | PEP-621 metadata rule; manifests dispatched by type |
| Go project scanning (go.mod) + LICENSE check | ✅ Done | module/go directives; repo self-scan 100/100 |
| `verax init` starter-suite generator | ✅ Done | live server → suite skeleton (commented call cases, green-first); verified against official server-memory (9 tools) |
| verax test -j N parallel suites | ✅ Done | bounded pool, order-stable reports; 4 suites live 27/27 | honeyport + string-arg injection; e2e on official everything (17 tools, 0 hits) |
| rest-api-wrap template | ✅ Done | zero-dep OpenAPI subset + offline demo harness; suite 7/7 (path/body/404/allowlist) |
| web-fetch-md template + result.not_contains | ✅ Done | fixed-port offline harness; scripts/styles leak check; suite 7/7, scan 100 |
| db-readonly template (JSONL tables) | ✅ Done | equals/contains filters, no SQL/eval; suite 8/8, scan 100 |
| next templates | ⬜ Next | 6 shipped |

## M1 addendum: HTTP targets (2026-11-08 evening)

| Feature | Status | Notes |
|---|---|---|
| S13 Streamable HTTP transport for test suites | ✅ Done | POST subset (JSON+SSE), fixture HTTP mode, url-target e2e green |

- 2026-11-08 (evening 3): Mcp-Session-Id echo; --json report; CI real-package scan (F-4: official server-memory, 6 advisories).

- 2026-11-08 — **v0.4.0 released**: session-echo, --json/--badge/--repeat,
  real-package CI scan (F-4: official server-memory, 6 advisories).

- 2026-11-08 — **v0.5.0 released**: side-effect probe, -j parallelism,
  pyproject/go.mod scanning, verax init, self-scan 100/100.

### Day-1 final tally / 首日总账 (2026-11-08 20:00)

- 75 commits · 5 releases (v0.1.0–v0.5.0) · 17 scanner rules · 6 templates
- `verax` CLI: test (8 flags) / scan (6 flags) / init / new / rules
- 45/45 suite cases green in parallel (-j 7); self-scan 100/100
- Field evidence F-1..F-4 against official reference servers
