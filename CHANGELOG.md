# Changelog

All notable changes. Dates are 2026-11-08 (initial development day).

## v0.12.0
- registry-pulse scans crates.io-packaged registry entries alongside npm
  (`--sample-crates`, default 5); pulse rows carry a `registry` field,
  markdown gains a Registry column
- npm/crate pools deduped independently by highest version; shared
  dedupe/count helpers extracted with tests

## v0.11.0
- `verax scan crate:name[@version]`: scan published crates.io artifacts —
  pure-HTTP fetch with an identifying User-Agent (crates.io policy),
  latest-version resolution, crate-root-stripping extraction reusing the
  escape-proof tar reader
- Action self-test now scans a real pinned crate (`tinyjson@2.5.1`)

## v0.10.0
- Rust rule pack: `verax scan` understands Rust projects (Cargo.toml +
  .rs) — 6 new source rules (VX-SEC-003 dynamic/shell spawn, VX-SEC-002
  computed fs write, VX-SEC-008 outbound URL, VX-SEC-014 unsafe code,
  VX-QUA-107 unwrap/expect panic surfaces, VX-PRV-303 credentials in
  logs) plus Cargo.toml metadata hygiene
- 18 rules total (js/py/rs/pkg); `testfixtures/scanrust` e2e in the
  action self-test

## v0.9.0
- `verax report`: pulse data → ecosystem report (markdown + self-contained
  HTML) — score distribution, severity mix, most prevalent rules, riskiest
  packages, per-package trend (keys survive version bumps)
- Wired into the nightly loop (regenerated after each pulse, published to
  Pages, cross-linked with the leaderboard)
- First live edition (2026-11-08 pulse): 29 packages, mean 83.6, top
  ecosystem risk VX-SEC-008 (107 hits)

## v0.8.1
- Weekly full-registry walk: Monday 05:33 Beijing cron + `full-walk`
  workflow dispatch input (150-pkg sample, `--max-pages` bounded)
- Nightly loop now closes itself: pulse → leaderboard regen → badge
  smoke → commit → Pages deploy trigger (previously manual steps)
- `verax registry-pulse`: page-fetch retry (3× linear backoff, non-2xx
  aware) and automatic `pulse-latest.json` write (badge service data)
- Fuzz corpus: 3 new categories — `sql-injection`, `json-rpc`, `xss`
  (8 total)

## v0.8.0
- `verax scan --fail-level low|medium|high|critical` severity gate
- Leaderboard: trend sparklines (SVG), pulse retry/backoff, Pages live
- Community fuzz corpus (`corpus/fuzz/*.txt`, embedded, PR-welcome,
  hygiene-enforced); registry-pulse `--all` full-registry walk

## v0.7.0
- `apps/badge`: badge service over pulse data (SVG, healthz, hourly reload)
- `apps/leaderboard` + `verax leaderboard`: pulses → static self-contained
  HTML leaderboard; nightly workflow regenerates after each pulse
- M4's visible loop closed end-to-end on live registry data

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
