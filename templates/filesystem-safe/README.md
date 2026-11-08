# filesystem-safe template

Verax MCP server template: filesystem access with root containment and
security defaults baked in — zero npm dependencies (clean supply chain by
construction).

## Use it

Copy `templates/filesystem-safe/` into your project, then:

```bash
VERAX_ROOT=/path/to/allowed/dir node index.js
```

Wire it into any MCP client (Claude Code, Cursor, ...) as a stdio server.

## Tools

| Tool | Does | Guard |
|---|---|---|
| `list_files` | list the allowed root (optional `sub`) | sub contained |
| `read_file` | read a text file | path contained |
| `write_file` | write a text file (mkdir -p parents) | path contained |

All paths resolve against `VERAX_ROOT`; anything escaping it is a tool-level
error (`path escapes allowed root`), never a crash and never a filesystem touch.

## Test it

The template ships with a Verax suite (containment, round trip, catalogue):

```bash
verax test tests/server.suite.yaml
```

## Score it

```bash
verax scan .          # expect 100/100 — that's the point of the template
```
