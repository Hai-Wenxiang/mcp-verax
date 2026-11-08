# Verax Scanner — `verax scan` / 静态扫描器

> **Status**: M2 core shipped (engine + 8 rules + SARIF + npm targets). Dynamic probes (VX-DYN-*) and more rule packs come next.
> **状态**：M2 核心已落地（引擎 + 8 条规则 + SARIF + npm 发布物扫描）。动态探针与更多规则包在路上。

## English

`verax scan` statically inspects an MCP server **as published** — a local
project directory, an npm spec, a crates.io crate (`crate:name[@version]`) or a PyPI sdist (`pypi:name[@version]`) — and reduces findings to a 0–100
score. It answers "is this server worth installing?", complementing
mcp-scan-style malicious-content detection.

### Quick start

```bash
verax scan ./my-mcp-server                     # local project
verax scan @scope/mcp-server                   # published artifact (npm pack)
verax scan crate:tinyjson                       # published artifact (crates.io)
verax scan pypi:mcp-server-fetch                 # published sdist (PyPI)
verax scan @scope/mcp-server --sarif out.json  # SARIF 2.1.0 for GitHub Security tab
verax scan . --fail-level medium          # exit 1 when findings reach a severity
verax scan . --badge b.svg --min-score 90 # SVG badge + score threshold
```

Exit codes: `0` clean / low-only · `1` any critical or high · `2` usage.

### Rule packs (18 rules, JS/TS + Python + Rust)

| Rule | Severity | Detects | Lang |
|---|---|---|---|
| VX-SEC-001 | critical | whole-environment exposure (bare `process.env`) | js |
| VX-SEC-013 | critical | unsafe deserialization (`pickle`/`yaml.load`/`marshal`) | py |
| VX-SEC-002 | high | filesystem writes to built paths, no containment markers | js |
| VX-SEC-003 | high | shell from non-literal input / `shell:true` / `os.system` | js+py |
| VX-SEC-004 | high | `eval`/`new Function` / `exec()` dynamic execution | js+py |
| VX-SEC-005 | high | credential-shaped literals (7 shapes, masked in output) | any |
| VX-SEC-011 | high/medium | dependency typosquats (edit distance 1/2 of popular pkgs) | pkg |
| VX-SEC-012 | medium | consumer-side install hooks (preinstall/install/postinstall) | pkg |
| VX-SEC-008 | medium | outbound requests to non-literal URLs without allowlist (SSRF) | js |
| VX-PRV-301 | medium | embedded telemetry endpoints (segment/mixpanel/posthog/…) | any |
| VX-PRV-302 | medium | hardcoded PII classifier patterns (email/phone/SSN regexes) | js |
| VX-QUA-103 | low | empty catch blocks (swallowed errors) | js |
| VX-QUA-105 | low/medium | package metadata completeness (missing manifest = medium) | pkg |
| VX-QUA-106 | low | README presence | pkg |

### Use in GitHub Actions

```yaml
- uses: Hai-Wenxiang/mcp-verax@main
  with:
    mode: scan
    target: @scope/mcp-server        # or a local directory
    sarif: verax.sarif
```

The action fails on critical/high findings and uploads the SARIF artifact.

Scoring: 100 − Σ weights (critical 25 / high 10 / medium 4 / low 1), floored at 0.

### Design notes

- Scans **the packed artifact** for npm targets (`npm pack`): you audit exactly
  what consumers install, not the repo's current state.
- Text-level heuristics with comment-line suppression; safe usages
  (`process.env.HOME`, literal `exec("git status")`, exact popular dep names)
  are deliberately excluded — fixtures carry these negatives inline.
- Tarball extraction rejects path escapes; symlinks skipped.

## 中文

`verax scan` 以"**发布物**"为对象静态审计 MCP server（本地目录或 npm 包名），输出 0–100 评分。回答的问题是"这个 server 值不值得装"——与 mcp-scan 式的恶意内容检测互补。

- 本地目录与 npm 包（`npm pack` 拉取后解包扫描，审的就是消费者装到的东西）两种目标；
- 8 条规则见上表（环境透传/命令拼接/动态执行/凭据字面量/typosquat/安装钩子/元数据/README）；
- 严重度加权扣分（25/10/4/1），触底为 0；critical/high 命中退出码 1，可做 CI 门禁；
- `--sarif` 输出 2.1.0 标准报告，可直接进 GitHub Security tab。

### 真实包验证（2026-11-08）

- `@modelcontextprotocol/server-memory` → 96/100（仅一条 prepare 备注）
- `@modelcontextprotocol/server-everything` → 71/100：`get-env` 工具整环境透传命中 VX-SEC-001（critical）——该 server 本就以暴露 env 为测试目的，规则在真实代码上有效。
