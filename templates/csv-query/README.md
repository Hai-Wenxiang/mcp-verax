# csv-query template

Verax MCP server template: query CSV files inside an allowed root, with hard
size/row caps and no expression evaluation — zero npm dependencies.

## Use it

```bash
VERAX_ROOT=/path/to/csvs node index.js
```

## Tools

| Tool | Does | Guard |
|---|---|---|
| `list_csv` | list CSVs in the root | contained |
| `describe_csv` | columns + row count (+truncation) | size/row caps |
| `query_csv` | substring filter on one column | no eval; result limit ≤100 |

## Test it

```bash
verax test tests/server.suite.yaml   # from repo root, see suite header
verax scan .                          # expect 100/100
```
