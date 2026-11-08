# ADR-001: Test Kit 实现语言 / Implementation Language for the Test Kit

- 状态 / Status: Accepted (2026-11-08)
- 决策人 / Decided by: AI（按 P-06「组件级选型、无全局主导」原则代理决策，用户可随时推翻 / decided under the per-component principle; user may veto anytime）
- 范围 / Scope: `verax` CLI 与 Test Kit 核心 runner（仅此组件，不约束其他组件）

## 决策 / Decision

**Test Kit 与 `verax` CLI 用 Go 实现。** 其余组件（模板、文档站、badge worker 等）开工时各自再写 ADR。

The Test Kit runner and `verax` CLI are implemented in **Go**. Every other component gets its own ADR when its work starts.

## 背景 / Context

- P-03 已确认暂不注册 npm → M1 分发渠道只有 GitHub Release；
- 协议面很薄：JSON-RPC 2.0 over stdio/HTTP，自实现约几百行，不依赖官方 SDK 的语言绑定；
- CI 需要三 OS 矩阵（win/mac/linux）与快速冷启动。

## 选项对比 / Options

| 选项 | 分发（无 npm） | 运行时依赖 | 交叉编译 | 备注 |
|---|---|---|---|---|
| **Go** ✅ | **单二进制挂 Release，`curl` 即用** | 无 | `GOOS/GOARCH` 一条命令 | 官方 SDK 非必需（协议薄） |
| TypeScript | `npx github:...`（可用但慢且需 Node） | Node 20+ | 需打包器 | 生态亲和最高，但与 P-03 冲突 |
| Python | pipx from git，需解释器 | Python 3.11+ | 麻烦 | pytest 插件形态佳，留作 F8 之后评估 |
| Rust | 单二进制 | 无 | 良好 | 迭代速度与安全权衡后不选 |

## 后果 / Consequences

- ✅ GitHub Action（F8）可直接下载二进制，矩阵测试秒级启动；
- ✅ 单二进制对「接入方验货」场景（在任意机器快速扫一个 server）体验最好；
- ⚠️ 仓库需同时容纳 Go（现在）与其他语言组件（将来），目录约定见 AGENTS.md；
- ⚠️ MCP 社区贡献者以 TS 居多，Go 贡献门槛略高——接受。
