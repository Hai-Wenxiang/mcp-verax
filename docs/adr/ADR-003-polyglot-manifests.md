# ADR-003: Polyglot manifests & green-first scaffolding / 三语清单与绿开箱

- 状态 / Status: Accepted (2026-11-08)
- 范围 / Scope: scanner manifests（package.json / pyproject.toml / go.mod）；`verax init`

## 决策 / Decision

1. 扫描器按**清单类型分发规则**：package.json 规则永不作用于 toml/go.mod，
   反之亦然；共用规则（README/LICENSE）在 scan 层统一判断。
   解析用务实子集扫描（无 TOML/YAML 依赖），不追求完整语法。
2. `verax init` 生成的 call 骨架**默认注释**：生成套件开箱即绿
   （discovery + leak），用户填好 TODO 再逐工具解开——拒绝"红开箱"。

## 理由 / Rationale

- MCP 生态三语并存（TS/Python/Go）；规则误跨类型会产生噪声（实证：早期
  JS 规则命中 .py 文件、json 规则误报 toml）。
- 骨架带着 TODO 参数必然失败（如 fetch_url 的 "TODO" URL）——注释掉是唯一
  同时满足"有用骨架"与"开箱即绿"的形态。

## 后果 / Consequences

- 新语言清单（如 Cargo.toml、gemspec）按同模式扩展：manifest 识别 → 类型
  守门 → 子集解析；
- init 用户需手动解注释，首次体验为"绿 → 改 → 可能红"，符合测试驱动直觉。
