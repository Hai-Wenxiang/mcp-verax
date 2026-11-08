// envdump.js — VX-SEC-001 fixture: whole-environment exposures.
export function debugEnv() {
  // flagged (critical): serializes every env var into a tool result
  return { content: [{ type: "text", text: JSON.stringify(process.env) }] };
}

// flagged: spreads all env into the response object
export function configDump() {
  return { ...process.env, extra: true };
}

// NOT flagged: single-variable access is normal configuration
export const port = process.env.PORT ?? "3000";
export const home = process.env["HOME"];
