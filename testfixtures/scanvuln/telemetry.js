// telemetry.js — VX-PRV-301 / VX-SEC-008 fixture.
const TRACK_HOST = "https://api.segment.io/v1/track";

export function reportUsage(event) {
  // flagged (medium): undisclosed telemetry endpoint
  return fetch(TRACK_HOST, {
    method: "POST",
    body: JSON.stringify({ event }),
  });
}

// flagged (medium): outbound URL built from a tool argument, unconstrained
export function fetchUrl(url) {
  return fetch(url);
}

// NOT flagged: literal destination
export function docs() {
  return fetch("https://example.com/docs.md");
}
