# AGENTS.md — Verax 仓库协作规范

> 本文件给 AI 协作者（以及未来的贡献者）提供仓库规则。配套方法论见 `docs/adr/` 与 `docs/STATUS.md`。

## 仓库地图

```
cmd/verax/          # CLI 入口
internal/           # Go 内部包（不对外 import）
  jsonrpc/          # JSON-RPC 2.0 编解码
  mcpclient/        # MCP 协议客户端：stdio/HTTP 传输、握手、原语
  dsl/              # 测试套件 YAML 解析与校验
  assert/           # 断言引擎与快照
  runner/           # 套件执行器
  fuzz/             # Schema 驱动模糊测试
  report/           # 终端/JUnit/HTML 报告器
testfixtures/       # Go 编写的 MCP 测试靶机（echo/crash/hang 等）
docs/               # 文档（双语：英文在前，中文对照）
  adr/              # 架构决策记录
  STATUS.md         # 功能进度总表（每次功能完成必须更新）
```

其他语言组件将来按 `templates/`（脚手架）、`apps/`（服务）等目录加入，各自带 ADR。

## 开发命令

```bash
export PATH=/e/MyProj01/tools/go/bin:$PATH   # 本机便携 Go（CI 用 setup-go）
go build ./...
go vet ./...
go test ./... -count=1
gofmt -l .                                      # 必须输出为空
go run ./cmd/verax version
```

- GOPROXY 走 `https://goproxy.cn,direct`（本机加速，CI 用默认）。

## 工作流（必须遵守）

1. **一个步骤一个 commit**：每完成一个可验证的小步骤（含测试）即 commit，再开始下一步。
2. **commit 信息**：Conventional Commits（`feat:`/`fix:`/`test:`/`docs:`/`chore:`），作者邮箱 `yinweikai2000@sina.com`（仓库级 git config 已配）。
3. **一个功能完成**：更新 `docs/STATUS.md` + 该功能文档（放 `docs/`，双语），然后 push。
4. **CI 绿灯**是下一个功能的开工条件（`.github/workflows/ci.yml` 三 OS 矩阵）。
5. **实时进展**只记在 `WORKLOG.md`（已 gitignore，**不入库**）。

## 代码约定

- Go ≥1.27；标准库优先，新增依赖需在任务说明中写明理由；
- 错误信息小写开头、带上下文（`fmt.Errorf("spawn server: %w", err)`）；
- 测试与实现同包或 `_test` 包，优先表驱动测试；
- 对外（文档/README）双语：英文在前，中文对照在后；代码注释英文；
- 不要提交：`WORKLOG.md`、`bin/`、`dist/`、录制响应原文（含敏感字段）、任何密钥。

## 已知陷阱

- Windows 下 spawn 子进程的 stdio 测试：路径用 `filepath`，参数不要依赖 shell 展开；
- stdio 传输的换行按 `\n` 处理，Windows 目标进程输出 `\r\n` 时需 trim；
- `go test` 里再 `go run` 靶机需要工具链在 PATH（CI runner 满足）。
