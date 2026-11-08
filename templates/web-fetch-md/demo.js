#!/usr/bin/env node
// demo.js — offline harness for the web-fetch-md template: serves stub HTML
// pages on a FIXED local port (so embedded suites can reference URLs),
// spawns index.js allowlisted to 127.0.0.1, proxies stdio.
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
const require = createRequire(import.meta.url);
const http = require("node:http");
const { spawn } = require("node:child_process");

const PORT = Number(process.env.DEMO_PORT || 39871);

const PAGES = {
  "/docs": `<!doctype html><html><head><title>Docs</title><style>body{color:red}</style>
<script>alert("never returned")</script></head><body>
<h1>Verax Docs</h1><p>Serve pages as readable text.</p>
<ul><li>allowlisted hosts only</li><li>scripts stripped</li></ul>
</body></html>`,
  "/plain": "just plain text, no html here",
};

const server = http.createServer((req, res) => {
  const url = new URL(req.url, "http://x");
  if (url.pathname in PAGES) {
    res.writeHead(200, { "Content-Type": url.pathname === "/plain" ? "text/plain" : "text/html" });
    return res.end(PAGES[url.pathname]);
  }
  res.writeHead(404, { "Content-Type": "text/plain" });
  res.end("not found");
});

function listenWithRetry(attempt, cb) {
  server.once("error", (err) => {
    if (err.code === "EADDRINUSE" && attempt < 10) {
      server.close();
      setTimeout(() => listenWithRetry(attempt + 1, cb), 200);
    } else {
      cb(err);
    }
  });
  server.listen(PORT, "127.0.0.1", cb);
}

listenWithRetry(0, (err) => {
  if (err) {
    process.stderr.write(`demo: cannot listen on ${PORT}: ${err.message}\n`);
    process.exit(1);
  }
  const child = spawn(process.execPath, [fileURLToPath(new URL("./index.js", import.meta.url))], {
    env: { ...process.env, ALLOWED_HOSTS: process.env.ALLOWED_HOSTS ?? "127.0.0.1" },
    stdio: ["pipe", "inherit", "inherit"],
  });
  process.stdin.pipe(child.stdin);
  child.on("exit", (code) => {
    server.close();
    process.exit(code === null ? 1 : code);
  });
});

process.stdin.on("end", () => {
  setTimeout(() => process.exit(0), 300).unref();
});
