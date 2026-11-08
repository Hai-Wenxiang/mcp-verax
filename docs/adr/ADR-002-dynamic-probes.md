# ADR-002: Dynamic probes run inside the Test Kit, not the Scanner / 动态探针归属

- 状态 / Status: Accepted (2026-11-08)
- 范围 / Scope: SSRF 探针（VX-DYN-201）与副作用探针（VX-DYN-202）

## 决策 / Decision

动态探针作为 **Test Kit 套件块**（`probe.ssrf` / `probe.side_effect`）实现，
由 runner 在**已拉起的服务进程**上执行；`verax scan` 保持纯静态
（源码 + 发布物），不负责启动任何代码。

Dynamic probes live as suite blocks executed by the runner against a live
spawned server; `verax scan` stays purely static and never executes target code.

## 理由 / Rationale

- 扫描器的目标常是 npm 发布物（无运行命令）；探针的目标是行为（必须有进程）。
  两者输入域不同，强行合并会让 scan 变成"要求用户提供启动命令"的工具。
- 套件里已具备一切基建：传输（stdio/HTTP）、超时、进程树击杀、报告管线——
  探针只是多一类 CaseResult，复用全部门禁与产物。
- 安全：scan 不执行目标代码 = 扫描不可信包时零风险；探针只在用户显式写套件时运行。

## 后果 / Consequences

- SSRF/副作用检查需要用户写套件（`verax init` 生成骨架降低成本）；
- 未来若出现"沙箱内动态扫描"需求（rootless 容器），可新增 `verax probe`
  命令复用同一探针实现——决策不阻碍该路径。
