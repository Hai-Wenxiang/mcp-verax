# Changelog

All notable changes. Dates are 2026-11-08 (initial development day).

## v0.6.0
- `verax registry-pulse`: official MCP registry pull → sample scans → dated
  JSON+MD reports; nightly workflow commits pulses to docs/registry/
- First live pulse (60 entries, 2 npm packages) fixed a real rule FP:
  `env = process.env` DI defaults now medium review items, not critical

## v0.5.0
- `probe.side_effect`: dynamic honeypot-diff probe (write-while-read-only)
- `verax test -j N`: suites in parallel
- Scanner: Python (pyproject.toml/PEP-621) and Go (go.mod) project support;
  LICENSE presence; verax self-scans 100/100
- `verax init`: generate a starter suite from a live server (green-first
  skeletons, verified on official server-memory)

## v0.4.0
- `verax test --repeat N`: per-case stability table with flake detection
- `verax test -j N`: suites run concurrently (bounded pool, stable order)
- `verax scan --json` and `--badge`: machine-readable + SVG score badge
- HTTP targets: `Mcp-Session-Id` capture/echo (stateful servers)
- CI scans a real published package on every push — F-4: official
  `@modelcontextprotocol/server-memory` ships dependency ranges with 6
  advisories (see `compat/FINDINGS.md`)

## v0.3.0
- `verax new`: scaffold servers from templates embedded in the binary
- Templates: `filesystem-safe`, `csv-query`, `notes-store` (zero-dependency,
  suite-green, self-scan 100/100)
- `probe.ssrf`: dynamic SSRF detection via honeyport URL injection
  (verified against official server-everything: 17 tools, 0 hits)

## v0.2.0
- `verax scan`: 16 rules across js/py/pkg, severity scoring, SARIF 2.1.0,
  npm-artifact targets (`npm pack`), osv.dev CVE enrichment, `--min-score`
  CI gate
- `verax rules` catalog + bilingual rule handbook (`docs/rules.md`)
- Test Kit: Streamable HTTP url-target suites (POST subset: JSON + SSE)
- Compatibility findings on official reference servers (`compat/FINDINGS.md`):
  2 Verax bugs found+fixed by dogfooding; official filesystem server
  Windows hang isolated with a reproducer

## v0.1.0
- `verax test`: declarative YAML suites — discovery/call assertions, golden
  snapshots, latency SLOs, process-leak & hang detection, schema-driven
  fuzzing; terminal/JUnit/HTML reports; composite GitHub Action; 5-platform
  release pipeline
