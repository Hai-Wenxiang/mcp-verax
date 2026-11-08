// harness-safe.js — negative fixture: child-process harness patterns that
// must NOT fire SEC-001/SEC-003 (spawn argv-array + env inheritance).
import { spawn } from "node:child_process";

export function launchServer(script, extraEnv) {
  // env spread goes into a spawn options object: inheritance, not exposure
  const child = spawn(process.execPath, [script, "--stdio"], {
    env: { ...process.env, ...extraEnv },
    stdio: ["pipe", "inherit", "inherit"],
  });
  return child;
}
