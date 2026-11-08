# db-readonly template

Verax MCP server template: read-only queries on JSONL table files — no SQL
engine, no expressions, zero npm dependencies. "SELECT" is a column filter.

## Use it

```bash
VERAX_ROOT=/path/to/tables node index.js
```

## Tools

| Tool | Does | Guard |
|---|---|---|
| `list_tables` | list `.jsonl` tables | contained |
| `describe_table` | columns + row count | size/row caps |
| `query` | equals/contains filter on one column | no eval; result cap 100 |

## Test it

```bash
verax test tests/server.suite.yaml   # from repo root, see header
verax scan .                          # expect 100/100
```
