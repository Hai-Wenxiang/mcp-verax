// shell.js — VX-SEC-003 fixture: command construction surfaces.
import { exec, execSync, spawn } from "node:child_process";

// flagged: template-built command
export function listFiles(dir) {
  exec(`ls -la ${dir}`);
}

// flagged: concatenation + shell:true double whammy (one finding per line)
export function convert(file) {
  execSync("ffmpeg -i " + file + " out.mp3", { shell: true });
}

// NOT flagged: literal command, args array, no shell
export function gitStatus(cwd) {
  spawn("git", ["status"], { cwd });
}
