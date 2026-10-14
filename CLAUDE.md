# Project rules

Open-source, self-hosted enterprise password manager. Requirements: [docs/requirements.md](docs/requirements.md) is the source of truth; cite requirement IDs (e.g. `ACC-04`, `SES-02`) in plans, commits and tests.

Area rules load on top of this file: [server/CLAUDE.md](server/CLAUDE.md) (Go back end), [web/CLAUDE.md](web/CLAUDE.md) (React web app), [extension/CLAUDE.md](extension/CLAUDE.md) (browser extension).

## 1. Workflow
- Don't start coding until the user says so. Planning and documentation tasks end by handing back for review.
- Never guess business logic or edge cases. Check docs/requirements.md first; if it is still unclear, ask concise questions before coding.
- For changes touching more than 2 files or introducing a new pattern, outline the plan first: files, approach, risks, requirement IDs covered.
- Keep diffs minimal and incremental. Don't refactor unrelated working code.
- If a request contradicts docs/requirements.md, stop and ask. When the user agrees to a behaviour change, update docs/requirements.md in the same change.
- When done, state what was implemented, which requirement IDs it covers, how to verify it (exact commands), and anything left undone.

## 2. Security non-negotiables
This is a password manager: a security bug is a product failure. These rules override convenience.
- Plaintext secrets, master passwords and unwrapped keys never leave the client. The server only stores and returns ciphertext and wrapped keys (section 9).
- No custom cryptography. Use only the primitives listed in docs/requirements.md section 13. Any change to crypto code updates the shared test vectors in `spec/` and passes in both Go and TypeScript.
- Never log, print, persist unencrypted, or send to analytics: passwords, master passwords, keys, session keys, tokens, invite links, TOTP secrets, ciphertext bodies, or PII. Log IDs, not emails.
- Never put secrets or PII in URLs, query strings, route params or error messages.
- Deny by default. Every endpoint is authenticated (except the SES-07 allowlist), session-encrypted, scoped to one organization, and permission-checked in the use-case layer.
- Compare tokens and hashes in constant time.
- Every security-relevant action writes an audit event (section 10).
- No new dependency in crypto, auth or session code without the user's explicit approval.

## 3. Code quality (all languages)
- Comments are single-line and explain *why*, never *what*. Exception: Go doc comments on exported identifiers.
- No `any` (TypeScript) and no `any` / `interface{}` (Go) in payloads, API responses, props or domain types. Define explicit types.
- Remove unused imports, dead code, commented-out code, `console.log` and debug prints.
- Extract magic numbers and strings (limits, durations, keys) into named constants.
- Naming follows each language: TypeScript camelCase for variables/functions, PascalCase for components/types; Go MixedCaps (exported) and mixedCaps (unexported); JSON fields camelCase; SQL snake_case.

## 4. Architecture
- Organize by feature/domain, with small single-responsibility files.
- Back end: clean architecture. Dependencies point inward: adapters (HTTP, Postgres, mail, jobs) → use cases → domain. Business rules and permission checks live in the domain and use cases, never in handlers or repositories. Details in server/CLAUDE.md.
- Front end: small reusable components; data and state logic in custom hooks.
- `spec/openapi.yaml` is the API contract. Change the spec first, then regenerate server stubs and the TypeScript client. Validation rules live in the spec and in one place per side.

## 5. Testing
- Back end: every new or changed domain rule and use case gets a focused test covering the happy path, the main error case, a permission-denied case, and (for org-scoped data) a cross-organization isolation case.
- Front end: no UI tests unless explicitly requested. Exceptions that always need unit tests: `packages/crypto`, the session-channel client, and the extension's URL matching, field detection and phishing guards.
- Mock external services, time and randomness. Crypto tests use fixed test vectors.

## 6. Reliability, API and data
- Never swallow errors. Structured back-end errors with correct HTTP status codes; graceful front-end fallbacks with loading, empty and error states.
- One error shape everywhere: `{"error": {"code": "grant_expired", "message": "...", "requestId": "..."}}`. `code` is a stable snake_case string clients can switch on.
- Validate all input (body, query, params) at the back-end boundary.
- Never hardcode secrets or URLs. Read them from validated env vars and document each one in `deploy/.env.example`.
- Every list endpoint is paginated (cursor-based; default 50, max 200).
- Parameterized queries only (sqlc). Never concatenate user input into SQL.
- Migrations are additive and include a Down section. Never drop or rename columns/tables without the user's confirmation; do it as expand → migrate data → contract across releases.
- Avoid N+1 queries.

## 7. Tooling
- Don't add dependencies unless clearly justified; say why in the plan. Pin versions. Never hand-edit lockfiles or `go.sum`.
- Use the project's scripts and Makefile targets.
- Run the checks listed in each area's CLAUDE.md after changes, or say clearly which ones you couldn't run.
- Update README, docs and `deploy/.env.example` when public APIs, env vars or setup steps change.
- Write documents as local Markdown in `docs/`. Diagrams as Mermaid.
- Leave no temp files or debug artifacts behind.
- Commit or push only when asked. Commit messages: `feat(grants): expiring grants (ACC-04)`.

## 8. Done checklist
- [ ] Requirement IDs met, and docs/requirements.md still matches the behaviour
- [ ] Security rules in section 2 hold (no secrets in logs, URLs or plaintext storage)
- [ ] Strict types, no dead code or debug output
- [ ] Input validation and error paths handled
- [ ] Change is minimal
- [ ] Back-end tests added (plus crypto/extension tests where section 5 requires them)
- [ ] Checks run and passing, or the gaps stated
