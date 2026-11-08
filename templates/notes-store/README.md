# notes-store template

Verax MCP server template: a contained notes store (list/read/write/delete)
with hard caps and sanitized ids — zero npm dependencies.

## Use it

```bash
VERAX_ROOT=/path/to/notes node index.js
```

## Tools

| Tool | Does | Guard |
|---|---|---|
| `list_notes` | list note ids | contained |
| `read_note` | read one note | id sanitized |
| `write_note` | create/update | note count + size caps |
| `delete_note` | delete | id sanitized |

## Test it

```bash
verax test tests/server.suite.yaml   # from repo root, see suite header
verax scan .                          # expect 100/100
```
