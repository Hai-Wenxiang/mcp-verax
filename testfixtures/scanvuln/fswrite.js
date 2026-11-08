// fswrite.js — VX-SEC-002 fixture: uncontained writes (no containment markers).
import { writeFileSync, appendFile } from "node:fs";

export function saveReport(name, data) {
  // flagged (high): computed path, no root check anywhere in this file
  writeFileSync("/tmp/" + name + ".json", data);
}

// NOT flagged: literal destination (the injection surface is built paths)
export function log(line) {
  appendFile("/var/log/mcp.log", line, () => {});
}
