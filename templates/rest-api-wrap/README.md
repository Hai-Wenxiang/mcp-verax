# rest-api-wrap template

Verax MCP server template: wrap any OpenAPI-described REST API as MCP tools —
zero npm dependencies, destination-allowlisted, ops-capped.

## Use it

```bash
OPENAPI_URL=https://api.example.com/openapi.json \
ALLOWED_HOSTS=api.example.com \
node index.js
```

`demo.js` runs a fully offline example (stub API + spec + this wrapper):

```bash
node demo.js    # then speak MCP on stdin, or:
verax test tests/server.suite.yaml   # from the repo root
```

## Guards

| Guard | Default | Why |
|---|---|---|
| ALLOWED_HOSTS | (empty = deny all) | SSRF: tools cannot steer requests anywhere |
| VERAX_MAX_OPS | 30 | context-window bombs from huge specs |
| response body | 4 KiB truncated | tool-result size sanity |
| request timeout | 5s | hangs become clean tool errors |

## Test it

```bash
verax test tests/server.suite.yaml   # offline via demo.js harness
verax scan .                          # expect 100/100
```
