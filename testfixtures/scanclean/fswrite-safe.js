// fswrite-safe.js — negative fixture: containment discipline present.
import { writeFileSync } from "node:fs";
import path from "node:path";

const allowedRoot = "/data/workspace";

export function save(name, data) {
  // containment: resolve inside allowedRoot and verify
  const target = path.resolve(allowedRoot, name);
  if (!target.startsWith(allowedRoot + path.sep)) {
    throw new Error("path escape");
  }
  writeFileSync(target, data);
}
