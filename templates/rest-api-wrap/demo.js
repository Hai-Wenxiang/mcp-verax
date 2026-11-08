#!/usr/bin/env node
// demo.js — offline harness for the rest-api-wrap template.
// Serves a stub API + its OpenAPI spec on a local port, then spawns
// index.js wired to that spec and proxies stdio. Lets the embedded Verax
// suite run fully offline; doubles as a working usage example.
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
const require = createRequire(import.meta.url);
const http = require("node:http");
const { spawn } = require("node:child_process");

const server = http.createServer((req, res) => {
  const url = new URL(req.url, "http://x");
  const send = (code, obj) => {
    res.writeHead(code, { "Content-Type": "application/json" });
    res.end(JSON.stringify(obj));
  };
  if (req.method === "GET" && url.pathname === "/openapi.json") {
    return send(200, {
      openapi: "3.0.0",
      info: { title: "demo-users", version: "1.0.0" },
      servers: [{ url: `http://127.0.0.1:${server.address().port}` }],
      paths: {
        "/users": {
          get: { operationId: "list_users", summary: "List users", parameters: [{ name: "limit", in: "query", schema: { type: "number" } }] },
          post: { operationId: "create_user", summary: "Create a user", requestBody: { content: { "application/json": { schema: { type: "object", properties: { name: { type: "string" }, role: { type: "string" } }, required: ["name"] } } } } },
        },
        "/users/{id}": {
          get: { operationId: "get_user", summary: "Get one user", parameters: [{ name: "id", in: "path", required: true, schema: { type: "string" } }] },
        },
      },
    });
  }
  if (req.method === "GET" && url.pathname === "/users") {
    return send(200, [{ id: 1, name: "alice" }, { id: 2, name: "bob" }]);
  }
  if (req.method === "GET" && /^\/users\/[^/]+$/.test(url.pathname)) {
    return url.pathname.endsWith("/1") ? send(200, { id: 1, name: "alice" }) : send(404, { error: "not found" });
  }
  if (req.method === "POST" && url.pathname === "/users") {
    let body = "";
    req.on("data", (c) => (body += c));
    req.on("end", () => send(201, { created: JSON.parse(body || "{}") }));
    return;
  }
  send(404, { error: "no route" });
});

server.listen(0, "127.0.0.1", () => {
  const port = server.address().port;
  const child = spawn(process.execPath, [fileURLToPath(new URL("./index.js", import.meta.url))], {
    env: {
      ...process.env,
      OPENAPI_URL: `http://127.0.0.1:${port}/openapi.json`,
      ALLOWED_HOSTS: process.env.ALLOWED_HOSTS ?? "127.0.0.1",
    },
    stdio: ["pipe", "inherit", "inherit"],
  });
  process.stdin.pipe(child.stdin);
  child.on("exit", (code) => {
    server.close();
    process.exit(code === null ? 1 : code);
  });
});

process.stdin.on("end", () => {
  // verax closes our stdin; give the child a beat to drain, then stop.
  setTimeout(() => process.exit(0), 300).unref();
});
