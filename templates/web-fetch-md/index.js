#!/usr/bin/env node
// web-fetch-md — Verax MCP server template: fetch allowlisted web pages as
// readable text, zero npm dependencies.
//
// Security defaults:
//   1. Destinations locked to ALLOWED_HOSTS (comma-separated).
//   2. Response size capped (VERAX_MAX_BYTES, default 512 KiB) and timeout 10s.
//   3. HTML stripped to text server-side; scripts/styles never returned.
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const readline = require("node:readline");

const ALLOWED = new Set(
  (process.env.ALLOWED_HOSTS || "").split(",").map((h) => h.trim().toLowerCase()).filter(Boolean)
);
const MAX_BYTES = Number(process.env.VERAX_MAX_BYTES || 512 * 1024);
const TIMEOUT_MS = Number(process.env.VERAX_HTTP_TIMEOUT_MS || 10000);

// Minimal, forgiving HTML → text: strips script/style blocks and tags,
// decodes the common entities. Not a browser — deliberately lossy.
function htmlToText(html) {
  return html
    .replace(/<script[\s\S]*?<\/script>/gi, " ")
    .replace(/<style[\s\S]*?<\/style>/gi, " ")
    .replace(/<!--[\s\S]*?-->/g, " ")
    .replace(/<(br|\/p|\/div|\/h[1-6]|\/li)\s*\/?>/gi, "\n")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/[ \t]+/g, " ")
    .replace(/\n\s*\n\s*\n+/g, "\n\n")
    .trim();
}

async function fetchPage(urlStr) {
  let u;
  try {
    u = new URL(urlStr);
  } catch {
    throw new Error(`invalid url: ${urlStr}`);
  }
  if (!ALLOWED.has(u.hostname.toLowerCase())) {
    throw new Error(`host not allowed: ${u.hostname} (allowlist: ${[...ALLOWED].join(", ") || "(empty)"})`);
  }
  const resp = await fetch(u, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
    redirect: "follow",
    headers: { "User-Agent": "verax-web-fetch-md/0.1 (+template)" },
  });
  const ct = resp.headers.get("content-type") || "";
  const raw = await resp.text();
  if (raw.length > MAX_BYTES) throw new Error(`response exceeds cap: ${MAX_BYTES} bytes`);
  const text = ct.includes("html") ? htmlToText(raw) : raw;
  return `status ${resp.status} ${u.hostname}\n\n${text.slice(0, MAX_BYTES / 2)}`;
}

const tools = [
  {
    name: "fetch_page",
    description: "Fetch an allowlisted page and return readable text.",
    inputSchema: {
      type: "object",
      properties: { url: { type: "string" } },
      required: ["url"],
    },
  },
];

function handle(msg) {
  const respond = (result) => ({ jsonrpc: "2.0", id: msg.id, result });
  switch (msg.method) {
    case "initialize":
      return respond({
        protocolVersion: (msg.params && msg.params.protocolVersion) || "2025-06-18",
        capabilities: { tools: { listChanged: false } },
        serverInfo: { name: "web-fetch-md", version: "0.1.0" },
        instructions: `hosts: ${[...ALLOWED].join(", ") || "(none)"}`,
      });
    case "notifications/initialized":
      return null;
    case "ping":
      return respond({});
    case "tools/list":
      return respond({ tools });
    case "tools/call": {
      const { name, arguments: a = {} } = msg.params || {};
      if (name !== "fetch_page") {
        return respond({ content: [{ type: "text", text: `tool not found: ${name}` }], isError: true });
      }
      return fetchPage(String(a.url || ""))
        .then((text) => respond({ content: [{ type: "text", text }] }))
        .catch((e) => respond({ content: [{ type: "text", text: String(e.message || e) }], isError: true }));
    }
    default:
      if (msg.id !== undefined && msg.id !== null) {
        return { jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: `method not found: ${msg.method}` } };
      }
      return null;
  }
}

const pending = [];
const rl = readline.createInterface({ input: process.stdin });
rl.on("line", (line) => {
  line = line.trim();
  if (!line) return;
  let msg;
  try { msg = JSON.parse(line); } catch { return; }
  const out = handle(msg);
  if (out && typeof out.then === "function") {
    pending.push(out.then((o) => o && process.stdout.write(JSON.stringify(o) + "\n")));
  } else if (out) {
    process.stdout.write(JSON.stringify(out) + "\n");
  }
});
rl.on("close", async () => {
  await Promise.allSettled(pending);
  process.exit(0);
});
