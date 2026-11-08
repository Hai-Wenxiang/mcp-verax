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
| S10+ Dynamic probes (VX-DYN), deps CVE (OSV), badge service | ⬜ Next | |

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

## M1 addendum: HTTP targets (2026-11-08 evening)

| Feature | Status | Notes |
|---|---|---|
| S13 Streamable HTTP transport for test suites | ✅ Done | POST subset (JSON+SSE), fixture HTTP mode, url-target e2e green |
