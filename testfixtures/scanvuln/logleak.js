// logleak.js — VX-PRV-303 fixture.
export function boot() {
  // flagged (medium): env value into stdout logs
  console.log("starting with key:", process.env.API_KEY);
}

export function fail(req) {
  // flagged (medium): auth header into logs
  console.error("request failed", req.headers.authorization);
}

// NOT flagged: ordinary logging
export function info() {
  console.log("server ready on port", process.env.PORT || 3000);
  logger.info("nothing sensitive here");
}
