# Verax Compatibility Findings — Official MCP Reference Servers / 兼容性发现

> **Run**: 2026-11-08, verax v0.1.0+, Windows 10 (node v24.19.0, npm 12), official npm packages.
> **Suites**: `compat/*.suite.yaml` — runnable via `verax test compat/*.suite.yaml` from the repo root.
> **Score**: everything 7/7 · memory 4/4 · sequential-thinking 4/4 · filesystem: catalogue/fuzz/leak only (see F-2).

## English

Four official reference servers were driven with real Verax suites (discovery,
calls, schema errors, SLO, fuzz, leak). Verax itself had **two bugs found and
fixed** by this exercise; one server-side bug was isolated with a reproducer.

### V-1 (Verax, fixed): discovery assumed all three primitives

`discover()` called `tools/list`, `resources/list` and `prompts/list`
unconditionally and failed the whole case on the first `-32601`. Official
servers legitimately implement subsets (filesystem/memory have no prompts).
**Fix**: capability-gated discovery + `-32601` tolerated as "primitive absent"
(commit `feat(runner): capability-gated discovery`).

### V-2 (Verax, fixed): handshake failures hid server stderr

A server that dies during `initialize` (crash, bad config, npx 404) left only
"read: EOF" — the actual reason sat invisibly on stderr. **Fix**: handshake
errors now carry the last server stderr lines (same commit).

### F-1 (server-everything 2.2.1): tool renames vs. common docs

`add` → `get-sum`, `longRunningOperation` → `trigger-long-running-operation`;
17 tools total. Suites targeting documented-but-renamed tools get a crisp
`Tool add not found` (-32631). Tip: a deliberately-impossible
`tools.list.includes` fails with a detail line listing the server's real
catalogue (`compat/probe.howto.yaml` documents this probe trick).

### F-2 (server-filesystem 2.2.0, Windows): rapid sequential calls hang

- `list_directory` (`.` or absolute path): **hangs indefinitely**.
- `read_file`: works in isolation/slow pacing, **hangs** when sent
  immediately after `tools/list`+`resources/list` responses.
- With 1–3s spacing between identical requests: **all four respond**.
- The hang is **non-deterministic and spreading across request kinds**: later
  runs tripped the path-escape probe and even a fuzz campaign that had passed
  twice before — any rapid-fire call sequence is at risk on Windows.
- Handshake, `tools/list`, and stdin-EOF shutdown behave correctly
  throughout; compat suite therefore keeps catalogue+leak checks only.

Reproducer (batch → hang, expect 3 responses and no 4th):

```bash
printf '%s\n' \
 '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"p","version":"0"}}}' \
 '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
 '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
 '{"jsonrpc":"2.0","id":3,"method":"resources/list"}' \
 '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"read_file","arguments":{"path":"README.md"}}}' \
 | npx -y "@modelcontextprotocol/server-filesystem" .
```

Environment: Windows 10, node v24.19.0. Suspected stdin-batching/pacing
sensitivity in the server's stdio transport on Windows. Not reproducible
against our fixture or server-everything with identical request patterns.

### F-3 (npx gotcha, tooling): hyphenated package segments need quoting

`npx -y @modelcontextprotocol/server-sequential-thinking` silently drops
`-thinking` as an npx flag → 404 → instant exit (surfaced as handshake EOF,
now with npm's stderr thanks to V-2). Quote the name:
`npx -y "@modelcontextprotocol/server-sequential-thinking"`.

### F-4 (server-memory 2.1.0): published with vulnerable dependency ranges

`verax scan @modelcontextprotocol/server-memory` (2026-11-08, osv.dev):
- `@modelcontextprotocol/server ^2.3.1` — 4 advisories
  (GHSA-46jr-4wx8-v453, GHSA-f897-62hf-6gxq, GHSA-jm63-cq47-j5jp, GHSA-rvq5-wwqv-78pq)
- `zod ^4.2.0` — 2 advisories (GHSA-m95q-7qp3-xv42, GHSA-mrhx-c7xc-m3w5)

Score 80/100. Not malicious — just stale ranges; exactly the kind of signal
"worth installing?" scans exist for. This scan now runs in CI on every push.

## 中文

用真实套件（发现/调用/参数错误/SLO/fuzz/泄漏）驱动四个官方参考 server。这次自举 **为 Verax 抓出并修复 2 个 bug**，隔离 1 个服务器端 bug（附复现）。

- **V-1（已修）**：discovery 假设三大原语齐全 —— 官方 filesystem/memory 本就没有 prompts 能力，-32601 不应炸掉整个用例。已改为按握手能力门控 + 容忍 method-not-found。
- **V-2（已修）**：握手失败只报 "read: EOF"，服务器死因（崩溃/配置错/npx 404）全在 stderr 里看不到。现在错误信息携带服务器 stderr 尾部。
- **F-1**：server-everything 2.2.1 工具改名（`add`→`get-sum` 等）；"不可能的 includes" 失败明细可直接 dump 真实工具清单（探针技巧见 probe.howto.yaml）。
- **F-2**：官方 filesystem 2.2.0 在 Windows 上：`list_directory` 无限挂起；`read_file` 紧跟 tools/list+resources/list 发送时挂起，间隔 1–3 秒重发则全部正常——疑似 stdio 传输对请求节奏敏感（服务器端问题，同模式打我们的 fixture 与 everything 均正常）。
- **F-3**：npx 会把包名里 `-thinking` 当自己的旗标吃掉 → 404 秒退。包名加引号即可。
