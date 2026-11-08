# Verax

> **Truth layer for the MCP ecosystem** — engineering-grade testing, security and quality tooling for MCP servers.
> **MCP 生态的"求真层"** —— 为 MCP server 提供工程级的测试、安全与质量工具链。
>
> *Registry discovers. Verax verifies.* / 注册表负责发现，Verax 负责证明。

[![CI](https://github.com/Hai-Wenxiang/mcp-verax/actions/workflows/ci.yml/badge.svg)](https://github.com/Hai-Wenxiang/mcp-verax/actions/workflows/ci.yml)
[![Release](https://img.shields.io/badge/release-v0.1.0-blue)](https://github.com/Hai-Wenxiang/mcp-verax/releases/tag/v0.1.0)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](./LICENSE)

## English

Verax is a toolkit built around one idea: an MCP server should be able to **prove** it is protocol-compliant, stable and safe — and its consumers should be able to verify that claim cheaply, in CI.

**Try it now** (single binary, no runtime deps):

```bash
# Linux/macOS/Windows binaries: https://github.com/Hai-Wenxiang/mcp-verax/releases
verax test your-server.suite.yaml --html report.html
```

**Components (M1 → M4):**

| Component | What it does | Status |
|---|---|---|
| **Verax Test Kit** (`verax test`) | Declarative YAML test suites for MCP servers: handshake/primitive assertions, result snapshots, schema-driven fuzzing, latency SLOs, process-leak detection. Ships as a single Go binary. | ✅ v0.1.0 released |
| **Verax Scanner** (`verax scan`) | 16 rules (js/py/pkg) + osv.dev CVEs → SARIF, 0–100 score, `--min-score` CI gate. | ✅ v0.2.0 |
| **Verax Templates** (`verax new`) | 5 zero-dep scaffolds — filesystem-safe, csv-query, notes-store, rest-api-wrap, web-fetch-md — each suite-green and self-scan 100/100. | 🚧 M3 |
| **Verax Registry Intelligence** | Registry mirror + batch quality pipeline → leaderboard & quarterly ecosystem reports. | M4 |

Docs index: [`docs/`](./docs/README.md) — [STATUS](./docs/STATUS.md) · [test-kit](./docs/test-kit.md) · [scanner](./docs/scanner.md) · [rules](./docs/rules.md) · [findings](./compat/FINDINGS.md). Contributing: [`AGENTS.md`](./AGENTS.md).

## 中文

Verax 围绕一个理念构建：MCP server 应当能**自证**协议合规、稳定与安全，而使用者应当能在 CI 里低成本地验证这个说法。

**规划组件（M1 → M4）：**

| 组件 | 作用 | 状态 |
|---|---|---|
| **Verax Test Kit**（`verax test`） | 声明式 YAML 测试套件：握手/原语断言、结果快照、Schema 驱动模糊测试、延迟 SLO、进程泄漏检测。以 Go 单二进制分发。 | 🚧 M1 开发中 |
| **Verax Scanner** | 安全/质量/隐私三类规则包 + 动态探针 → SARIF、0–100 评分、徽章。 | M2 |
| **Verax Templates** | 内置测试与安全默认值的 server 脚手架（多语言）。 | M3 |
| **Verax Registry Intelligence** | 注册表镜像 + 批量质检流水线 → 榜单与季度生态报告。 | M4 |

设计文档：[`docs/STATUS.md`](./docs/STATUS.md)（进展）、[`docs/adr/`](./docs/adr/)（决策）、[`AGENTS.md`](./AGENTS.md)（贡献指南）。

---

## License / 许可证

[Apache-2.0](./LICENSE) © 2026 The Verax Authors
