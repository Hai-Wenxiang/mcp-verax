#!/usr/bin/env node
// rest-api-wrap — Verax MCP server template: wrap an OpenAPI-described REST
// API as MCP tools, zero npm dependencies, allowlist-enforced.
//
// Security defaults:
//   1. Destinations locked to ALLOWED_HOSTS (comma-separated host names).
//   2. Operations capped (MAX_OPS, default 30) — no context-window bombs.
//   3. Responses truncated (4 KiB) and calls time-boxed (5s).
// Spec source: OPENAPI_URL (fetched) or OPENAPI_FILE (local path).
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const fs = require("node:fs");
const path = require("node:path");
const readline = require("node:readline");

const MAX_OPS = Number(process.env.VERAX_MAX_OPS || 30);
const ALLOWED = new Set(
  (process.env.ALLOWED_HOSTS || "").split(",").map((h) => h.trim().toLowerCase()).filter(Boolean)
);
const BASE_TIMEOUT_MS = Number(process.env.VERAX_HTTP_TIMEOUT_MS || 5000);

function slug(op, method, p) {
  if (op.operationId) return op.operationId;
  return method + "_" + p.replace(/[^a-zA-Z0-9]+/g, "_").replace(/^_|_$/g, "");
}

function buildTools(spec) {
  const base = (spec.servers && spec.servers[0] && spec.servers[0].url) || "";
  const tools = [];
  for (const [p, item] of Object.entries(spec.paths || {})) {
    for (const method of ["get", "post", "put", "patch", "delete"]) {
      const op = item[method];
      if (!op) continue;
      if (tools.length >= MAX_OPS) return { base, tools };

      const props = {};
      const required = [];
      for (const param of op.parameters || []) {
        props[param.name] = { type: param.schema && param.schema.type === "number" ? "number" : "string" };
        if (param.required) required.push(param.name);
      }
      const bodySchema =
        op.requestBody && op.requestBody.content &&
        op.requestBody.content["application/json"] && op.requestBody.content["application/json"].schema;
      for (const [name, sch] of Object.entries((bodySchema && bodySchema.properties) || {})) {
        props[name] = { type: sch.type === "number" ? "number" : sch.type === "boolean" ? "boolean" : "string" };
      }
      tools.push({
        name: slug(op, method, p),
        description: op.summary || `${method.toUpperCase()} ${p}`,
        inputSchema: { type: "object", properties: props, required },
        _meta: { method, path: p },
      });
    }
  }
  return { base, tools };
}

async function callTool(base, tool, args) {
  let url = base + tool._meta.path.replace(/\{([^}]+)\}/g, (_, k) => encodeURIComponent(String(args[k] ?? "")));
  const u = new URL(url);
  if (!ALLOWED.has(u.hostname.toLowerCase())) {
    throw new Error(`host not allowed: ${u.hostname} (allowlist: ${[...ALLOWED].join(", ") || "(empty)"})`);
  }
  const method = tool._meta.method.toUpperCase();
  const init = { method, signal: AbortSignal.timeout(BASE_TIMEOUT_MS), headers: {} };
  if (method !== "GET" && method !== "DELETE") {
    init.headers["Content-Type"] = "application/json";
    const body = {};
    for (const k of Object.keys(tool.inputSchema.properties)) {
      if (!tool._meta.path.includes("{" + k + "}") && args[k] !== undefined) body[k] = args[k];
    }
    init.body = JSON.stringify(body);
  } else {
    for (const [k, v] of Object.entries(args)) {
      if (!tool._meta.path.includes("{" + k + "}")) u.searchParams.set(k, String(v));
    }
    url = u.toString();
  }
  const resp = await fetch(url, init);
  const text = (await resp.text()).slice(0, 4096);
  return `status ${resp.status}\n${text}`;
}

async function main() {
  let specText;
  if (process.env.OPENAPI_FILE) {
    specText = fs.readFileSync(process.env.OPENAPI_FILE, "utf8");
  } else if (process.env.OPENAPI_URL) {
    const r = await fetch(process.env.OPENAPI_URL, { signal: AbortSignal.timeout(10000) });
    specText = await r.text();
  } else {
    process.stderr.write("rest-api-wrap: set OPENAPI_URL or OPENAPI_FILE\n");
    process.exit(1);
  }
  const spec = JSON.parse(specText);
  const { base, tools } = buildTools(spec);

  const handle = (msg) => {
    const respond = (result) => ({ jsonrpc: "2.0", id: msg.id, result });
    switch (msg.method) {
      case "initialize":
        return respond({
          protocolVersion: (msg.params && msg.params.protocolVersion) || "2025-06-18",
          capabilities: { tools: { listChanged: false } },
          serverInfo: { name: "rest-api-wrap", version: "0.1.0" },
          instructions: `wrapping ${base} (${tools.length} ops, hosts: ${[...ALLOWED].join(", ")})`,
        });
      case "notifications/initialized":
        return null;
      case "ping":
        return respond({});
      case "tools/list":
        return respond({ tools: tools.map(({ _meta, ...t }) => t) });
      case "tools/call": {
        const { name, arguments: a = {} } = msg.params || {};
        const tool = tools.find((t) => t.name === name);
        if (!tool) return respond({ content: [{ type: "text", text: `tool not found: ${name}` }], isError: true });
        return callTool(base, tool, a)
          .then((text) => respond({ content: [{ type: "text", text }] }))
          .catch((e) => respond({ content: [{ type: "text", text: String(e.message || e) }], isError: true }));
      }
      default:
        if (msg.id !== undefined && msg.id !== null) {
          return { jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: `method not found: ${msg.method}` } };
        }
        return null;
    }
  };

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
}

main().catch((e) => {
  process.stderr.write(`rest-api-wrap: ${e.message}\n`);
  process.exit(1);
});
