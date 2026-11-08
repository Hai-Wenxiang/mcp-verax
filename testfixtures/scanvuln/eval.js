// eval.js — VX-SEC-004 fixture: dynamic evaluation surfaces.
const userToolInput = "1+1";

// flagged: direct eval of tool-controlled data
export function calc(expr) {
  return eval(expr);
}

// flagged: Function constructor
export function compile(src) {
  return new Function("return " + src);
}

// NOT flagged: comment mentioning eval(
// eval("this line is a comment")
export const safe = 41 + 1;
