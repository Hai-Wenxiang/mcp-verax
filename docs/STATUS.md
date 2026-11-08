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
| F5 SLO (latency p95, concurrency) + process-leak + hang detection | ⬜ Planned | |
| F6 Schema-driven fuzzing + injection dictionaries | ⬜ Planned | |
| F7 Reporters (JUnit XML, static HTML) | ⬜ Planned | |
| F8 GitHub Action + release pipeline | ⬜ Planned | |
| F9 Bootstrap E2E + first compatibility report | ⬜ Planned | stretch |

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
