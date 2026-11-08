# Verax Rule Handbook / 规则手册

> All scanner rules with fix guidance. Live catalog: `verax rules`.
> 全部扫描规则与修复建议；命令行实时目录：`verax rules`。

| Rule | Sev | Lang | Detects | Fix / 修复 |
|---|---|---|---|---|
| VX-SEC-001 | critical | js | whole-environment exposure (bare `process.env`) | Return only the keys you need: `pick(process.env, ["MODE"])` / 只返回所需键 |
| VX-SEC-002 | high | js+rs | fs writes to built paths, no containment | Resolve against a root and reject escapes (`path.resolve` + `startsWith`) / 限制在根目录内并拒绝逃逸 |
| VX-SEC-003 | high | js+py+rs | shell from non-literal input / `shell:true` / `os.system` / `Command::new(dynamic)` | Use argv-array APIs (`spawn(cmd, [args])`, `subprocess.run([...])`) / 用参数数组 API |
| VX-SEC-004 | high | js+py | `eval` / `new Function` / `exec()` | Replace with explicit parsers or lookup tables / 用显式解析或查表替代 |
| VX-SEC-005 | high | any | credential-shaped literals | Move to env/secret store; rotate leaked keys; add repo scanning / 改走环境变量并轮换已泄漏密钥 |
| VX-SEC-006 | high | pkg | dependencies with known advisories (osv.dev) | Upgrade the flagged versions / 升级到修复版本 |
| VX-SEC-008 | medium | js+rs | outbound to non-literal URL, no allowlist | Validate scheme+host against an explicit allowlist / 显式白名单校验目标 |
| VX-SEC-011 | high/med | pkg | typosquat dependencies (edit distance 1/2) | Verify the exact name against the registry / 核对包名拼写 |
| VX-SEC-012 | medium | pkg | consumer install hooks (pre/install/post) | Remove install-time execution; run build steps pre-publish / 移除安装期执行 |
| VX-SEC-013 | critical | py | unsafe deserialization (`pickle`/`yaml.load`/`marshal`) | `yaml.safe_load`; JSON for untrusted data / 改用 safe_load 或 JSON |
| VX-SEC-014 | medium | rs | `unsafe` blocks / impls | Each block is a manual audit item; isolate and shrink it / 逐块人工审计并尽量缩小 |
| VX-PRV-301 | medium | any | embedded telemetry endpoints | Remove or make opt-in and documented / 移除或改为显式可选 |
| VX-PRV-302 | medium | js | hardcoded PII classifiers (email/phone/SSN) | Disclose data handling in README / 在文档中声明数据处理行为 |
| VX-PRV-303 | medium | js+rs | credentials written to logs | Log identifiers, not values / 记录标识符而非值 |
| VX-QUA-103 | low | js | empty catch blocks | Handle or rethrow; never swallow silently / 处理或上抛 |
| VX-QUA-105 | low/med | pkg | incomplete package metadata | Fill license/repository/description / 补全元数据 |
| VX-QUA-106 | low | pkg | missing README | Document usage and intent / 补 README |
| VX-QUA-107 | low | rs | `unwrap()`/`expect()` panic surfaces | Return `Result` errors per call instead of crashing the server / 每次调用返回错误而非 panic |

Scoring: `100 − Σ weights`（critical 25 / high 10 / medium 4 / low 1，触底 0）。CI 门禁：critical/high 命中或 `--min-score N` 不达标即退出码 1。
