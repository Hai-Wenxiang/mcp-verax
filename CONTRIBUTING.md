# Contributing to Verax / 贡献指南

Thanks for helping make MCP servers provable. / 感谢帮助"MCP server 可被证明"这件事成真。

## The one rule / 唯一铁律

Every change lands as small verified steps:

```
one step (with tests) → commit → docs when a feature completes
→ push → CI green (3-OS matrix + action self-test) → next step
```

`AGENTS.md` documents the repo map and conventions — read it first.

## Ways to help / 贡献方式

| Area | Where | How |
|---|---|---|
| **Scanner rules** | `internal/scanner/rule_*.go` | One rule = ID + detection + fixtures (positive AND negative) + test. Ship both directions or it doesn't ship. |
| **Templates** | `templates/<id>/` | Zero npm dependencies; embedded Verax suite green; self-scan 100/100. `verax new` picks it up automatically (embedded). |
| **Fuzz dictionary** | `internal/fuzz/dictionary.go` | Inert payloads only — never live URLs or real credentials (see the hygiene test). |
| **Compat suites** | `compat/*.suite.yaml` | Suites for third-party servers + findings to `compat/FINDINGS.md`. Reproducers required. |
| **Docs** | `docs/` | Bilingual (EN first, 中文对照). `docs/rules.md` must stay in sync with the rule registry (`verax rules`). |

## Adding a scanner rule — checklist / 加规则清单

1. `VX-<SEC|QUA|PRV|DYN>-<NNN>` ID, severity by weight table (25/10/4/1);
2. detection regex/code — RE2 **has no lookaheads**, check context in code;
3. fixtures: positive example in `testfixtures/scanvuln*`, negative in `scanclean` (⚠ fixture comments must not contain trigger words — they will suppress the rule!);
4. test asserting both directions + counts synced in `TestVulnScoreDrops`;
5. `docs/rules.md` row with fix guidance (EN + 中文).

## Local dev / 本地开发

```bash
go build ./... && go test ./... -count=1   # the gate everything else depends on
go run ./cmd/verax test examples/mcp-fixture.suite.yaml -v
go run ./cmd/verax scan testfixtures/scanclean --no-osv   # must be 100/100
```

## License

Apache-2.0. Contributions land under the same license.
