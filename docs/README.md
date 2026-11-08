# Verax Docs / 文档索引

> Single entry point for all Verax documentation. / Verax 全部文档的入口。

| Doc | What's inside | 内容 |
|---|---|---|
| [STATUS.md](./STATUS.md) | Progress board across milestones | 功能进度总表 |
| [test-kit.md](./test-kit.md) | `verax test`: DSL reference (assertions, snapshots, SLO, leak, fuzz, SSRF probe, HTTP targets), watch mode, reports | 测试套件使用与 DSL 参考 |
| [scanner.md](./scanner.md) | `verax scan`: rule packs, scoring, SARIF, badges, OSV CVEs, npm-artifact targets | 扫描器与规则包 |
| [rules.md](./rules.md) | Rule handbook with fix guidance (bilingual) | 规则手册与修复建议 |
| [adr/](./adr/) | Architecture decision records | 架构决策记录 |
| [../compat/FINDINGS.md](../compat/FINDINGS.md) | Official reference server compatibility findings | 官方 server 兼容性发现 |

## Templates / 模板

`verax new <template> <name>` scaffolds from the binary-embedded library
(`verax new` with no args lists them):

| Template | Does | Guards |
|---|---|---|
| `filesystem-safe` | contained file access | root containment, escape errors |
| `csv-query` | query CSVs | size/row caps, no expression eval |
| `notes-store` | notes CRUD | id charset, count/byte caps |
| `rest-api-wrap` | OpenAPI → tools | ALLOWED_HOSTS lock, op cap, timeouts |
| `web-fetch-md` | fetch pages as text | ALLOWED_HOSTS, size caps, script strip |
| `db-readonly` | query JSONL tables | equals/contains only, caps |

Every template ships zero npm dependencies, an embedded Verax suite
(green in CI on every push) and scans 100/100.

## GitHub Actions

```yaml
- uses: Hai-Wenxiang/mcp-verax@v0.3.0
  with: { mode: test, files: tests/*.suite.yaml, junit: v.xml }
- uses: Hai-Wenxiang/mcp-verax@v0.3.0
  with: { mode: scan, target: @scope/mcp-server, sarif: v.sarif }
```

## CLI surface / 命令一览

```
verax test  [-u] [-v] [--watch] [--junit x.xml] [--html x.html] <suite.yaml>...
verax scan  [--sarif x.json] [--badge x.svg] [--min-score N] [--no-osv] <dir | @scope/pkg>
verax rules
verax new   <template> <name> [--dir base]
```
