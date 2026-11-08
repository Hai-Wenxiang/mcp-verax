# Verax Test Kit — `verax test` / 测试套件运行器

> **Status**: M1 core shipped (F1–F4). Fuzz/SLO/leak/reporters/action follow (F5–F8).
> **状态**：M1 核心已落地（F1–F4）；fuzz/SLO/泄漏检测/报告器/Action 见 F5–F8。

## English

`verax test` runs declarative YAML test suites against a real MCP server (spawned over stdio) and reports PASS/FAIL per case with assertion details — in your terminal today, in CI via exit codes, with JUnit/HTML reporters coming next.

### Quick start

From the repo root (Go toolchain required):

```bash
go run ./cmd/verax test examples/mcp-fixture.suite.yaml
```

Or build once:

```bash
go build -o verax ./cmd/verax
verax test path/to/your.suite.yaml -v
```

### Suite file reference

```yaml
suite: my-server                      # required name
protocol: 2025-06-18                  # optional: 2025-06-18 | 2025-03-26 | 2024-11-05
target:
  command: npx -y "@scope/mcp-server" # stdio launch (quotes honoured) — or:
  # url: https://remote.example.com   # remote Streamable HTTP target
  # headers: {authorization: "Bearer ${TOKEN}"}
  env: {API_TOKEN: "${MY_TOKEN}"}     # ${VAR} expanded from environment
  timeout_startup_ms: 15000           # spawn/connect + initialize budget (default 15000)
before:                               # probes before cases
  - initialize.ok
case:
  # Discovery case: no `call`, asserts the server's catalogues.
  - name: tools catalogue
    expect:
      tools.list.includes: [create_issue, list_prs]   # all must exist
      tools.list.exact: [create_issue, list_prs]      # exact set match
      tools.list.count: 2
      tools.list.schema_valid: true                   # every inputSchema is an object
      resources.list.min: 1
      prompts.list.includes: [summarize]

  # Call case: invokes a tool, asserts on the outcome.
  - name: create issue happy path
    call: create_issue
    args: {owner: octocat, repo: hello, title: t}
    env: {GH_FAKE_API: "fixture://github.json"}   # per-case env → server respawned
    timeout_ms: 5000
    expect:
      ok: true                          # no error, isError=false
      is_error: false                   # (either form)
      error.contains: "rate limited"    # matches isError content OR RPC error text
      result.contains: ["created"]      # substrings in content text
      result.snapshot: issue-created    # golden snapshot (see below)
```

Execution model / 执行模型:

- **Shared mode** (default): one long-lived server process for the whole suite — fast, realistic.
- **Respawn mode**: any case with `env:` forces a fresh server per case (fault injection isolation).
- **HTTP mode**: `target.url` suites run against a remote Streamable HTTP server (POST subset: JSON + SSE responses, session header echo); per-case env and process-leak checks do not apply remotely.

Exit codes: `0` all pass · `1` any failure · `2` usage error.

### Snapshots

`result.snapshot: <name>` stores the canonical pretty-JSON of the tool result in
`__snapshots__/<suite>.snap.json` (commit it). First sighting records and passes;
afterwards results must match byte-for-byte. Accept changes with:

```bash
verax test suite.yaml -u    # rewrite goldens (review the diff!)
```

### What's checked today / roadmap

| Capability | Status |
|---|---|
| Handshake (initialize + version negotiation) | ✅ |
| tools/resources/prompts discovery assertions | ✅ |
| tools/call assertions (ok/is_error/contains) | ✅ |
| Golden snapshots + `-u` | ✅ |
| Latency SLO (p95, concurrency) | ✅ F5 |
| Process-leak & hang detection | ✅ F5 |
| Schema-driven fuzzing | ✅ F6 |
| JUnit XML + HTML reports | ✅ F7 |
| GitHub Action | ✅ F8 |

### Use in GitHub Actions

```yaml
- uses: Hai-Wenxiang/mcp-verax@main
  with:
    files: tests/*.suite.yaml
    junit: verax-junit.xml
    html: verax-report.html
```

The action fails the job on any suite failure and emits reports as artifacts.
Until the first binary release, it builds the CLI from source with Go (~10s).

### Fuzz block

```yaml
fuzz:
  runs: 100          # total hostile calls (seeded, reproducible)
  seed: 42
  hang_ms: 5000      # per-call hang budget
  tools: [echo]      # default: all tools
```

Tool-level error results (`isError`) are tolerated; transport/RPC failures
count as **crashes**, budget overruns as **hangs**. The first crash aborts the
campaign (a dead server would inflate every subsequent call into a crash).

### SLO block

```yaml
slo:
  tool: echo           # tool to hammer (required when budgets are set)
  args: {msg: "hi"}
  p95_call_ms: 800     # latency budget
  concurrency: 4       # parallel callers
  sample_calls: 20     # total samples (default 20)
  success_rate: 1.0
```

### Leak block

```yaml
leak:
  process_exit: true   # server must exit on stdin EOF within grace (3s)
```

### SSRF probe (dynamic)

```yaml
probe:
  ssrf: {hang_ms: 4000}
```

Starts a local honeyport, injects its URL into every string argument of
every tool, and fails the suite naming each tool that made the server dial
out — the SSRF surface static rules can only guess at. Verified against the
official server-everything (17 tools, 0 hits) and a deliberately fetching
fixture (hit reported as `fetch_url.url`).

## 中文

`verax test` 用声明式 YAML 套件对真实 MCP server（stdio 拉起）做回归测试：断言/快照/延迟 SLO/进程泄漏/Schema 模糊测试一应俱全，终端 + JUnit XML + 单文件 HTML 三种报告，`--junit/--html` 落盘，GitHub Action 三行接入 CI。

### 快速开始

```bash
go run ./cmd/verax test examples/mcp-fixture.suite.yaml   # 仓库根目录
```

### 套件文件要点

- `target.command`：stdio 拉起命令（支持引号）；`env` 支持 `${VAR}` 展开（未定义即报错，防止 CI 静默出错）；
- **发现类用例**（无 `call`）：断言 tools/resources/prompts 目录（includes/exact/count/schema_valid/min）；
- **调用类用例**（有 `call`）：断言 `ok` / `is_error` / `error.contains`（同时匹配工具错误内容与 JSON-RPC 错误文本）/ `result.contains` / `result.snapshot`；
- 带 `env:` 的用例会触发**重生模式**（每用例新起 server，故障注入互不污染）；
- 快照存于 `__snapshots__/<suite>.snap.json`，首次记录即通过，之后逐字节比对，`-u` 接受变更。

### 退出码

`0` 全过 · `1` 有失败 · `2` 用法错误。

### 设计文档

- 协议层/传输层/断言引擎的实现细节见源码 `internal/`（jsonrpc / mcpclient / dsl / assert / runner / report）；
- 语言选型见 [ADR-001](./adr/ADR-001-test-kit-language.md)；
- 进度总表见 [STATUS.md](./STATUS.md)。
