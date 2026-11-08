# Verax Fuzz Corpus

Hostile string payloads used by `verax test` fuzz blocks (schema-driven
injection into tool arguments). **Contributions welcome via PR** — one
payload per line, blank lines and `#` comments ignored.

## Rules / 规则

1. **Inert payloads only** — never live URLs, real credentials, or anything
   that exfiltrates. The CI hygiene test rejects them.
2. One payload per line in the category files below.
3. New categories = new `.txt` file; the loader picks them up automatically.

## Files

| File | Category |
|---|---|
| `path-traversal.txt` | filesystem escape attempts |
| `shell-injection.txt` | shell/code injection surfaces |
| `prompt-injection.txt` | instructions aimed at the model |
| `encoding.txt` | unicode/control/structural junk |
| `size.txt` | oversize and boundary strings |
