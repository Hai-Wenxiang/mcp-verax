#!/usr/bin/env node
// notes-store — Verax MCP server template: a contained notes store with
// hard size caps, zero npm dependencies.
//
// Security defaults:
//   1. All notes contained to VERAX_ROOT (resolve + prefix check).
//   2. Hard caps: MAX_NOTES and MAX_NOTE_BYTES — inputs cannot blow memory.
//   3. Note ids are sanitized ([a-z0-9-_]); no path tricks.
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const fs = require("node:fs");
const path = require("node:path");
const readline = require("node:readline");

const ROOT = path.resolve(process.env.VERAX_ROOT || "./data");
const MAX_NOTES = Number(process.env.VERAX_MAX_NOTES || 1000);
const MAX_NOTE_BYTES = Number(process.env.VERAX_MAX_NOTE_BYTES || 64 * 1024);

function notePath(id) {
  if (!/^[a-z0-9-_]{1,64}$/i.test(id)) {
    throw new Error("invalid note id (allowed: letters, digits, -, _; max 64)");
  }
  const target = path.resolve(ROOT, id + ".md");
  if (target !== ROOT && !target.startsWith(ROOT + path.sep)) {
    throw new Error(`path escapes allowed root: ${id}`);
  }
  return target;
}

function index() {
  if (!fs.existsSync(ROOT)) return [];
  return fs.readdirSync(ROOT)
    .filter((f) => f.toLowerCase().endsWith(".md"))
    .map((f) => f.slice(0, -3))
    .sort();
}

const tools = [
  { name: "list_notes", description: "List note ids.", inputSchema: { type: "object", properties: {} } },
  {
    name: "read_note",
    description: "Read one note.",
    inputSchema: { type: "object", properties: { id: { type: "string" } }, required: ["id"] },
  },
  {
    name: "write_note",
    description: "Create or update a note.",
    inputSchema: {
      type: "object",
      properties: { id: { type: "string" }, content: { type: "string" } },
      required: ["id", "content"],
    },
  },
  {
    name: "delete_note",
    description: "Delete a note.",
    inputSchema: { type: "object", properties: { id: { type: "string" } }, required: ["id"] },
  },
];

function callTool(name, args) {
  switch (name) {
    case "list_notes":
      return index().join("\n") || "(no notes)";
    case "read_note":
      return fs.readFileSync(notePath(args.id), "utf8");
    case "write_note": {
      const ids = index();
      const target = notePath(args.id);
      const exists = fs.existsSync(target);
      if (!exists && ids.length >= MAX_NOTES) {
        throw new Error(`note cap reached (${MAX_NOTES})`);
      }
      const content = String(args.content);
      if (Buffer.byteLength(content, "utf8") > MAX_NOTE_BYTES) {
        throw new Error(`note exceeds cap: ${MAX_NOTE_BYTES} bytes`);
      }
      fs.mkdirSync(ROOT, { recursive: true });
      fs.writeFileSync(target, content);
      return (exists ? "updated " : "created ") + args.id;
    }
    case "delete_note": {
      const target = notePath(args.id);
      if (!fs.existsSync(target)) throw new Error(`no such note: ${args.id}`);
      fs.unlinkSync(target);
      return "deleted " + args.id;
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
        serverInfo: { name: "notes-store", version: "0.1.0" },
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
