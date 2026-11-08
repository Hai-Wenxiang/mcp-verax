// pii.js — VX-PRV-302 fixture: hardcoded personal-data classifiers.
const EMAIL_RE = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/;
const PHONE_RE = /\d{3}-\d{3}-\d{4}/;

export function scan(message) {
  return {
    emails: message.match(EMAIL_RE) || [],
    phones: message.match(PHONE_RE) || [],
  };
}
