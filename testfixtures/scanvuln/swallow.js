// swallow.js — VX-QUA-103 fixture.
export function risky(path) {
  try {
    return load(path);
  } catch {} // flagged (low): failure swallowed, caller sees success-ish noise
}

export function loud(path) {
  try {
    return load(path);
  } catch (e) {
    throw new Error("load failed: " + e.message); // NOT flagged: handled
  }
}
