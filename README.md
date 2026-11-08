# Verax

> **Truth layer for the MCP ecosystem** — engineering-grade testing, security and quality tooling for MCP servers.
> **MCP 生态的"求真层"** —— 为 MCP server 提供工程级的测试、安全与质量工具链。
>
> *Registry discovers. Verax verifies.* / 注册表负责发现，Verax 负责证明。

[![CI](https://github.com/Hai-Wenxiang/mcp-verax/actions/workflows/ci.yml/badge.svg)](https://github.com/Hai-Wenxiang/mcp-verax/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Hai-Wenxiang/mcp-verax)](https://github.com/Hai-Wenxiang/mcp-verax/releases/latest)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](./LICENSE)

## English

Verax is a toolkit built around one idea: an MCP server should be able to **prove** it is protocol-compliant, stable and safe — and its consumers should be able to verify that claim cheaply, in CI.

**Try it now** (single binary, no runtime deps):

```bash
# Linux/macOS/Windows binaries: https://github.com/Hai-Wenxiang/mcp-verax/releases
verax test your-server.suite.yaml --html report.html   # suites: assertions, SLO, fuzz, SSRF probe
verax test your-server.suite.yaml --repeat 10          # flake detection: per-case stability
verax scan @scope/mcp-server --badge badge.svg         # score a published package (osv CVEs incl.)
```

**Field evidence** (2026-11-08): scanning the *official* `@modelcontextprotocol/server-memory`
found 6 dependency advisories (see [compat/FINDINGS.md](./compat/FINDINGS.md) F-4); the SSRF probe
cleared official server-everything (17 tools, 0 outbound hits); five scaffold templates all
self-scan 100/100 — the flywheel (template → suite-green → scan-clean) runs end to end in CI.

**Components (M1 → M4):**

| Component | What it does | Status |
|---|---|---|
| **Verax Test Kit** (`verax test`) | Declarative YAML test suites for MCP servers: handshake/primitive assertions, result snapshots, schema-driven fuzzing, latency SLOs, process-leak detection. Ships as a single Go binary. | ✅ shipped |
| **Verax Scanner** (`verax scan`) | 16 rules (js/py/pkg) + osv.dev CVEs → SARIF, 0–100 score, `--min-score` / `--fail-level` CI gates. | ✅ shipped |
| **Verax Templates** (`verax new`) | 6 zero-dep scaffolds — filesystem-safe, csv-query, notes-store, rest-api-wrap, web-fetch-md, db-readonly — each suite-green and self-scan 100/100. | ✅ 6 templates · M3 productization next |
| **Verax Registry Intelligence** | Registry pulse (nightly sample + weekly full walk) → live leaderboard, score badges, trend lines, [ecosystem report](https://hai-wenxiang.github.io/mcp-verax/report.html). | 🚧 live · [leaderboard](https://hai-wenxiang.github.io/mcp-verax/) |

[Changelog](./CHANGELOG.md) · Docs index: [`docs/`](./docs/README.md) — [STATUS](./docs/STATUS.md) · [test-kit](./docs/test-kit.md) · [scanner](./docs/scanner.md) · [rules](./docs/rules.md) · [findings](./compat/FINDINGS.md). Contributing: [`CONTRIBUTING.md`](./CONTRIBUTING.md) · [`AGENTS.md`](./AGENTS.md).

**Live leaderboard**: [hai-wenxiang.github.io/mcp-verax](https://hai-wenxiang.github.io/mcp-verax/) — nightly registry pulse, verax-scored.

## 中文

Verax 围绕一个理念构建：MCP server 应当能**自证**协议合规、稳定与安全，而使用者应当能在 CI 里低成本地验证这个说法。

```bash
verax init tests/x.suite.yaml -- npx -y "@scope/mcp-server"   # 活 server 生成起始套件（开箱即绿）
verax test tests/x.suite.yaml --repeat 10 -j 4                # 断言/探针/SLO/泄漏/fuzz + 稳定性
verax scan @scope/mcp-server --badge b.svg --min-score 90      # 规则+CVE 评分门禁 + 徽章
verax new csv-query my-server                                 # 零依赖模板脚手架（套件绿、自扫满分）
```

**组件（M1 → M4）：**

| 组件 | 作用 | 状态 |
|---|---|---|
| **Verax Test Kit**（`verax test`） | 声明式 YAML 测试套件：握手/原语断言、结果快照、Schema 驱动模糊测试、延迟 SLO、进程泄漏检测。以 Go 单二进制分发。 | ✅ 已交付 |
| **Verax Scanner**（`verax scan`） | 安全/质量/隐私三类规则包 + osv.dev CVE → SARIF、0–100 评分、`--min-score` / `--fail-level` CI 门禁。 | ✅ 已交付 |
| **Verax Templates**（`verax new`） | 6 个零依赖脚手架——filesystem-safe、csv-query、notes-store、rest-api-wrap、web-fetch-md、db-readonly——套件绿、自扫满分。 | ✅ 6 模板 · M3 产品化待启 |
| **Verax Registry Intelligence** | 注册表脉冲（每夜抽样 + 每周全量）→ 线上榜单、评分徽章、趋势线。季度生态报告后续。 | 🚧 [线上榜单](https://hai-wenxiang.github.io/mcp-verax/) |

设计文档：[`docs/STATUS.md`](./docs/STATUS.md)（进展）、[`docs/adr/`](./docs/adr/)（决策）、[`AGENTS.md`](./AGENTS.md)（贡献指南）。

---

## License / 许可证

[Apache-2.0](./LICENSE) © 2026 The Verax Authors
