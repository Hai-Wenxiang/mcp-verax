# Verax

> **Truth layer for the MCP ecosystem** — 为 MCP 生态提供测试、安全与质量的工程化工具链。
> *Verax*（拉丁语："求真的"）—— 注册表负责发现 server，Verax 负责证明它安全可用。

<p align="center">
  <em>⚠️ 预开发阶段（Pre-development）：本仓库当前仅包含设计文档与骨架，尚未开始编码。</em>
</p>

## 规划中的四个子产品

| 产品 | 包名（规划） | 一句话 |
|---|---|---|
| **Verax Test Kit** | `verax-test-kit` | "vitest for MCP servers"：声明式 YAML 场景测试 + fuzz + SLO + 进程泄漏检测，CI 零成本回归 |
| **Verax Scanner** | `verax-scanner` | 安全 + 质量 + 隐私三类规则包的扫描平台，SARIF 输出与 0–100 评分徽章 |
| **Verax Templates** | `verax-templates` | 自带测试与安全默认值的 MCP server 脚手架（TS/Python 双语言） |
| **Verax Registry Intelligence** | — | 官方注册表镜像 + 批量质检流水线 → 徽章、榜单、季度生态报告 |

## 状态与文档

- 📋 [项目实施方案](../docs/Verax项目实施方案.md) —— 定名、架构、里程碑、环境准备记录
- 🔍 [方向调研报告](../docs/AI集成开源项目调研报告.md) —— 为什么选 MCP 生态工具集
- 📖 [MCP 工具集项目详解](../docs/MCP工具集项目详解.md) —— 产品与竞争格局深度展开

> 注：设计文档暂存于本仓库上级 `docs/` 目录，公开开发启动后会迁入本仓库 `docs/`。

## 路线图（摘要）

- [ ] **M1（第 1–4 周）**：Test Kit 0.1.0 —— DSL + runner + 断言/快照/fuzz/SLO/泄漏检测 + GitHub Action + 官方 server 兼容性报告
- [ ] **M2（第 5–10 周）**：Scanner v0 —— 规则引擎 + 安全/质量规则包 + SARIF + 首份《热门 server 质量报告》
- [ ] **M3（第 11–20 周）**：模板库 10+（TS）+ 动态探针
- [ ] **M4（第 21–36 周）**：Registry Intelligence —— 镜像、榜单站、季报自动化

## License

[Apache-2.0](./LICENSE) © 2026 The Verax Authors
