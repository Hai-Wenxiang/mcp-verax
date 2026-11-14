# Verax Skills

Agent skills for the Verax toolkit — SKILL.md files that teach coding
agents (Claude Code, Cursor, Codex CLI, Copilot, …) when and how to run
`verax`.

## `verax/`

The main skill: install command with pinned versions + SHA256 verification,
command routing (init/test/scan/new/rules), CI gates, output interpretation
and pitfalls.

### Install into an agent

| Agent | How |
|---|---|
| Claude Code (project) | copy `skills/verax/` to `.claude/skills/verax/` in the repo |
| Claude Code (user) | copy to `~/.claude/skills/verax/` |
| Generic (SKILL.md-compatible agents) | copy to the agent's skills directory (e.g. `~/.agents/skills/verax/`) |
| Agensi and other marketplaces | zip `skills/verax/` and follow the marketplace's listing flow |

After installing, ask your agent "test my MCP server" or "scan this MCP
server" — the skill's description routes those requests to Verax.

### Distribution

- The skill is Apache-2.0 like the rest of Verax; free to list anywhere.
- Keep one source of truth: this directory. Copy (don't fork content) when
  uploading to marketplaces, so `skills/skills_test.go` stays the gate.
- Marketplace listing (Agensi outreach 2026-11-08) is a manual step — see
  the maintainer checklist in `docs/STATUS.md`.
