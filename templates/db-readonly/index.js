#!/usr/bin/env node
// db-readonly — Verax MCP server template: read-only SQL queries against a
// JSON-line database file, zero npm dependencies, no SQL engine.
//
// Security defaults:
//   1. Only SELECT-like projections on a local .jsonl file — no SQL parsing,
//      no eval, no network: queries are column filters, not statements.
//   2. Row and byte caps; results limited.
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const fs = require("node:fs");
const path = require("node:path");
const readline = require("node:readline");

const ROOT = path.resolve(process.env.VERAX_ROOT || "./data");
const MAX_FILE_BYTES = Number(process.env.VERAX_MAX_FILE_BYTES || 8 << 20);
const MAX_ROWS = Number(process.env.VERAX_MAX_ROWS || 20000);

function contained(rel) {
  const target = path.resolve(ROOT, rel);
  if (target !== ROOT && !target.startsWith(ROOT + path.sep)) {
    throw new Error(`path escapes allowed root: ${rel}`);
  }
  return target;
}

function loadTable(rel) {
  const target = contained(rel);
  const st = fs.statSync(target);
  if (st.size > MAX_FILE_BYTES) throw new Error(`file exceeds cap: ${st.size} > ${MAX_FILE_BYTES} bytes`);
  const lines = fs.readFileSync(target, "utf8").split("\n").filter(Boolean).slice(0, MAX_ROWS);
  const rows = lines.map((l) => JSON.parse(l));
  return { rows, columns: rows.length ? Object.keys(rows[0]) : [] };
}

const tools = [
  { name: "list_tables", description: "List .jsonl tables in the root.", inputSchema: { type: "object", properties: {} } },
  {
    name: "describe_table",
    description: "Columns + row count of a table.",
    inputSchema: { type: "object", properties: { table: { type: "string" } }, required: ["table"] },
  },
  {
    name: "query",
    description: "Filter rows where a column equals or contains a value. No SQL, no expressions.",
    inputSchema: {
      type: "object",
      properties: {
        table: { type: "string" },
        column: { type: "string" },
        equals: { type: "string" },
        contains: { type: "string" },
        limit: { type: "number" },
      },
      required: ["table", "column"],
    },
  },
];

function callTool(name, args) {
  switch (name) {
    case "list_tables":
      return fs.readdirSync(ROOT).filter((f) => f.endsWith(".jsonl")).join("\n") || "(no tables)";
    case "describe_table": {
      const t = loadTable(args.table);
      return `columns: ${t.columns.join(", ")}\nrows: ${t.rows.length}`;
    }
    case "query": {
      const t = loadTable(args.table);
      if (!t.columns.includes(args.column)) {
        throw new Error(`unknown column: ${args.column}; have ${t.columns.join(", ")}`);
      }
      const limit = Math.min(Number(args.limit || 20), 100);
      let rows = t.rows;
      if (args.equals !== undefined) rows = rows.filter((r) => String(r[args.column]) === String(args.equals));
      if (args.contains !== undefined) rows = rows.filter((r) => String(r[args.column]).includes(String(args.contains)));
      return [t.columns.join(","), ...rows.slice(0, limit).map((r) => t.columns.map((c) => r[c]).join(","))].join("\n");
    }
    default:
      throw new Error(`tool not found: ${name}`);
  }
}

function handle(msg) {
  const respond = (result) => ({ jsonrpc: "2.0", id: msg.id, result });
  switch (msg.method) {
    case "initialize":
      return respond({
        protocolVersion: (msg.params && msg.params.protocolVersion) || "2025-06-18",
        capabilities: { tools: { listChanged: false } },
        serverInfo: { name: "db-readonly", version: "0.1.0" },
      });
    case "notifications/initialized":
      return null;
    case "ping":
      return respond({});
    case "tools/list":
      return respond({ tools });
    case "tools/call": {
      const { name, arguments: a = {} } = msg.params || {};
      try {
        return respond({ content: [{ type: "text", text: String(callTool(name, a)) }] });
      } catch (e) {
        return respond({ content: [{ type: "text", text: e.message }], isError: true });
      }
    }
    default:
      if (msg.id !== undefined && msg.id !== null) {
        return { jsonrpc: "2.0", id: msg.id, error: { code: -32601, message: `method not found: ${msg.method}` } };
      }
      return null;
  }
}

const rl = readline.createInterface({ input: process.stdin });
rl.on("line", (line) => {
  line = line.trim();
  if (!line) return;
  let msg;
  try { msg = JSON.parse(line); } catch { return; }
  try {
    const out = handle(msg);
    if (out) process.stdout.write(JSON.stringify(out) + "\n");
  } catch (e) {
    process.stderr.write(`handler error: ${e.message}\n`);
  }
});
