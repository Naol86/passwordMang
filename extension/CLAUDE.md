# Browser extension rules (WXT, Manifest V3, Chrome + Firefox)

Applies on top of the root [CLAUDE.md](../CLAUDE.md). TypeScript, React, styling and i18n rules from [web/CLAUDE.md](../web/CLAUDE.md) also apply to the popup and options pages.

## Architecture
- **Background service worker:** owns the session, the session key, the decrypted vault and all API calls.
- **Content scripts:** only detect fields and fill them on command. They never hold the vault; they receive one credential, for one fill, after the background has checked the origin.
- **Popup and options:** React + shadcn/ui. They talk to the background through messages, never to the API directly.
- Message types are defined once, in one shared file. Every handler checks `sender.id` and the sender tab's origin before acting.

## Autofill and phishing guards
- Origin matching follows EXT-07 and EXT-08. Parse URLs with the URL API and a public-suffix list; never compare URLs as strings.
- Never fill in a cross-origin iframe without user confirmation. Never fill an HTTPS-saved credential into an HTTP page.
- Autofill on page load is off by default (EXT-06) and only fills when exactly one item matches.
- Content scripts run in the isolated world only. No main-world injection, no `eval`, no remote code.

## Storage and lifecycle
- Session key and unlocked state live in `browser.storage.session` (memory-only). Never put keys or plaintext in `storage.local` or `storage.sync`.
- MV3 workers can be stopped at any time: rebuild state from `storage.session` on wake, and never rely on long-lived globals.
- Lock on idle and on browser close (EXT-01).

## Permissions
- Request the minimum. Each permission in the manifest has a one-line justification in the README.
- Prefer optional host permissions, requested when the user turns on autofill (EXT-06).

## Cross-browser
- Use WXT's `browser` API. Every change must build and work in both Chrome and Firefox.

## Tests
Unit tests are required for URL matching, field detection heuristics and phishing guards.

## Checks to run
```bash
npm run typecheck && npm run lint && npm run test && npx wxt build && npx wxt build -b firefox
```
