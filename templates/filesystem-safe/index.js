#!/usr/bin/env node
// filesystem-safe — Verax MCP server template with security defaults baked in.
//
// Zero npm dependencies by design: a template that ships no supply chain is
// clean by construction. Implements the MCP stdio surface directly:
// initialize / tools list / tools call over newline-delimited JSON-RPC.
//
// Security defaults (what this template demonstrates):
//   1. All file access is contained to VERAX_ROOT (resolve + prefix check).
//   2. Path escapes are rejected with a clear tool error, never a crash.
//   3. No eval, no shell, no network, no telemetry.
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const fs = require("node:fs");
const path = require("node:path");
const readline = require("node:readline");

const ROOT = path.resolve(process.env.VERAX_ROOT || "./data");

function contained(rel) {
  const target = path.resolve(ROOT, rel);
  if (target !== ROOT && !target.startsWith(ROOT + path.sep)) {
    throw new Error(`path escapes allowed root: ${rel}`);
  }
  return target;
}

const tools = [
  {
    name: "list_files",
    description: "List files in the allowed root (no recursion into escapes).",
    inputSchema: { type: "object", properties: { sub: { type: "string" } } },
  },
  {
    name: "read_file",
    description: "Read a text file inside the allowed root.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" } },
      required: ["path"],
    },
  },
  {
    name: "write_file",
    description: "Write a text file inside the allowed root.",
    inputSchema: {
      type: "object",
      properties: { path: { type: "string" }, content: { type: "string" } },
      required: ["path", "content"],
    },
  },
];

function callTool(name, args) {
  switch (name) {
    case "list_files": {
      const dir = args.sub ? contained(args.sub) : ROOT;
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      return entries.map((e) => (e.isDirectory() ? e.name + "/" : e.name)).join("\n") || "(empty)";
    }
    case "read_file":
      return fs.readFileSync(contained(args.path), "utf8");
    case "write_file": {
      const target = contained(args.path);
      fs.mkdirSync(path.dirname(target), { recursive: true });
      fs.writeFileSync(target, String(args.content));
      return `wrote ${args.path} (${String(args.content).length} bytes)`;
    }
    default:
      throw new Error(`tool not found: ${name}`);
  }
}

function handle(msg) {
  const isNotification = msg.id === undefined || msg.id === null;
  const respond = (result) => ({ jsonrpc: "2.0", id: msg.id, result });
  const respondErr = (code, message) => ({ jsonrpc: "2.0", id: msg.id, error: { code, message } });

  switch (msg.method) {
    case "initialize":
      return respond({
        protocolVersion: (msg.params && msg.params.protocolVersion) || "2025-06-18",
        capabilities: { tools: { listChanged: false } },
        serverInfo: { name: "filesystem-safe", version: "0.1.0" },
        instructions: `File access contained to ${ROOT}`,
      });
    case "notifications/initialized":
      return null;
    case "ping":
      return respond({});
    case "tools/list":
      return respond({ tools });
    case "tools/call": {
      const { name, arguments: args = {} } = msg.params || {};
      try {
        const text = callTool(name, args);
        return respond({ content: [{ type: "text", text: String(text) }] });
      } catch (e) {
        return respond({ content: [{ type: "text", text: e.message }], isError: true });
      }
    }
    default:
      if (!isNotification) return respondErr(-32601, `method not found: ${msg.method}`);
      return null;
  }
}

const rl = readline.createInterface({ input: process.stdin });
rl.on("line", (line) => {
  line = line.trim();
  if (!line) return;
  let msg;
  try {
    msg = JSON.parse(line);
  } catch {
    return;
  }
  try {
    const out = handle(msg);
    if (out) process.stdout.write(JSON.stringify(out) + "\n");
  } catch (e) {
    process.stderr.write(`handler error: ${e.message}\n`);
  }
});
