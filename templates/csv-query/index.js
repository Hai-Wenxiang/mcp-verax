#!/usr/bin/env node
// csv-query — Verax MCP server template: query CSV files inside an allowed
// root, zero npm dependencies.
//
// Security defaults:
//   1. All reads contained to VERAX_ROOT (resolve + prefix check).
//   2. Hard caps: MAX_ROWS and MAX_FILE_BYTES — inputs cannot blow memory.
//   3. Filtering is by simple column equality/contains — no expression eval.
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const fs = require("node:fs");
const path = require("node:path");
const readline = require("node:readline");

const ROOT = path.resolve(process.env.VERAX_ROOT || "./data");
const MAX_FILE_BYTES = Number(process.env.VERAX_MAX_FILE_BYTES || 5 << 20); // 5 MiB
const MAX_ROWS = Number(process.env.VERAX_MAX_ROWS || 5000);

function contained(rel) {
  const target = path.resolve(ROOT, rel);
  if (target !== ROOT && !target.startsWith(ROOT + path.sep)) {
    throw new Error(`path escapes allowed root: ${rel}`);
  }
  return target;
}

// Minimal RFC4180-ish CSV parser: quotes, doubled quotes, CRLF tolerated.
function parseCSV(text) {
  const rows = [];
  let row = [], field = "", inQ = false;
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (inQ) {
      if (c === '"') {
        if (text[i + 1] === '"') { field += '"'; i++; } else { inQ = false; }
      } else field += c;
      continue;
    }
    if (c === '"') { inQ = true; continue; }
    if (c === ",") { row.push(field); field = ""; continue; }
    if (c === "\n" || c === "\r") {
      if (c === "\r" && text[i + 1] === "\n") i++;
      row.push(field); field = "";
      if (row.length > 1 || row[0] !== "") rows.push(row);
      row = [];
      continue;
    }
    field += c;
  }
  row.push(field);
  if (row.length > 1 || row[0] !== "") rows.push(row);
  return rows;
}

function loadTable(rel) {
  const target = contained(rel);
  const st = fs.statSync(target);
  if (st.size > MAX_FILE_BYTES) throw new Error(`file exceeds cap: ${st.size} > ${MAX_FILE_BYTES} bytes`);
  const rows = parseCSV(fs.readFileSync(target, "utf8"));
  if (rows.length === 0) throw new Error("empty CSV");
  const header = rows[0].map((h) => h.trim());
  const data = rows.slice(1, 1 + MAX_ROWS).map((r) => Object.fromEntries(header.map((h, i) => [h, r[i] ?? ""])));
  return { header, data, truncated: rows.length - 1 > MAX_ROWS };
}

const tools = [
  {
    name: "list_csv",
    description: "List CSV files in the allowed root.",
    inputSchema: { type: "object", properties: {} },
  },
  {
    name: "describe_csv",
    description: "Show columns, row count and truncation state of a CSV.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
  {
    name: "query_csv",
    description: "Filter rows where column contains a substring. No expressions.",
    inputSchema: {
      type: "object",
      properties: {
        path: { type: "string" },
        column: { type: "string" },
        contains: { type: "string" },
        limit: { type: "number" },
      },
      required: ["path", "column"],
    },
  },
];

function callTool(name, args) {
  switch (name) {
    case "list_csv": {
      const entries = fs.readdirSync(ROOT, { withFileTypes: true });
      return entries.filter((e) => e.isFile() && e.name.toLowerCase().endsWith(".csv")).map((e) => e.name).join("\n") || "(no csv files)";
    }
    case "describe_csv": {
      const t = loadTable(args.path);
      return `columns: ${t.header.join(", ")}\nrows: ${t.data.length}${t.truncated ? ` (truncated at ${MAX_ROWS})` : ""}`;
    }
    case "query_csv": {
      const t = loadTable(args.path);
      if (!t.header.includes(args.column)) {
        throw new Error(`unknown column: ${args.column}; have ${t.header.join(", ")}`);
      }
      const limit = Math.min(Number(args.limit || 20), 100);
      const needle = String(args.contains ?? "");
      const rows = t.data.filter((r) => String(r[args.column]).includes(needle)).slice(0, limit);
      return [t.header.join(","), ...rows.map((r) => t.header.map((h) => r[h]).join(","))].join("\n");
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
        serverInfo: { name: "csv-query", version: "0.1.0" },
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
