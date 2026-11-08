# web-fetch-md template

Verax MCP server template: fetch allowlisted web pages and return readable
text — scripts/styles stripped server-side, zero npm dependencies.

## Use it

```bash
ALLOWED_HOSTS=docs.example.com,api.example.com node index.js
```

`demo.js` runs an offline example (stub pages on 127.0.0.1:39871 + wrapper).

## Guards

| Guard | Default | Why |
|---|---|---|
| ALLOWED_HOSTS | (empty = deny all) | SSRF: tools cannot steer fetches |
| VERAX_MAX_BYTES | 512 KiB | response bombs |
| timeout | 10s | hangs become clean tool errors |

## Test it

```bash
verax test tests/server.suite.yaml   # offline via demo.js
verax scan .                          # expect 100/100
```
