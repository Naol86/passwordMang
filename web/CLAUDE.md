# Front-end rules (React + TypeScript + Vite + Tailwind + shadcn/ui)

Applies on top of the root [CLAUDE.md](../CLAUDE.md).

## Layout
```text
src/app/                    router, providers, layouts, error boundaries
src/features/<feature>/     auth, companies, folders, items, grants, members, env, audit, settings
  components/               feature components
  hooks/                    data and state logic (useItems, useGrantForm, ...)
  api.ts                    calls through the generated API client
  types.ts                  feature types
src/components/ui/          shadcn/ui generated components; change only through theming
src/components/common/      shared app components
src/lib/                    small utilities (cn, formatters, constants)
```
Crypto and the session channel come from `packages/crypto` and `packages/api-client`. Never reimplement them in the app.

## TypeScript
- `strict: true` and `noUncheckedIndexedAccess: true`. No `any`, no `@ts-ignore`, no `!` non-null assertions without a one-line reason.
- API types come from the generated client. Don't cast API data with `as`; validate untrusted shapes with zod.
- Props are typed as `type <Component>Props = { ... }`.

## Components and state
- Function components, small and focused. Data fetching, mutations and business logic live in custom hooks, not in components.
- Server state through TanStack Query only. Local UI state with `useState` / `useReducer`. No global store unless agreed first.
- Forms use the shadcn Form pattern (react-hook-form + zod).
- Routes are company-scoped: `/o/:orgSlug/...` (ACT-04). Switching company resets company data (ACT-05).
- Every query and mutation handles loading, empty and error states. Each route has an error boundary; mutation failures show a toast mapped from the API error `code`.

## Styling and UX
- Tailwind utilities and shadcn components; merge classes with `cn()`. No inline styles.
- Colours, spacing and radii come from theme tokens (CSS variables). No hard-coded colours. Light and dark mode both work.
- Accessibility: WCAG 2.2 AA (NFR-10). Every input has a label, icon buttons have an `aria-label`, dialogs trap and return focus, everything works by keyboard.
- All user-facing text goes through the i18n function (NFR-11). No hard-coded strings in components.
- Virtualize lists that can exceed 100 rows. Code-split per route.

## Security
- Decrypted vault data and keys live in memory only. Never write keys or plaintext to localStorage, sessionStorage, IndexedDB or cookies.
- Clear decrypted data on lock, sign-out and idle timeout (SEC-13).
- Hidden fields are masked by default; copying clears the clipboard after 30 seconds (ITM-05).
- No `dangerouslySetInnerHTML`, no `eval`, no third-party scripts, CDNs or web fonts (OPS-08).
- Never put secrets or plaintext in URLs, React Query keys or error reports.

## Tests
None for UI unless the user asks. Code in `packages/crypto` and `packages/api-client` (session channel) always has unit tests.

## Checks to run
```bash
npm run typecheck && npm run lint && npm run build
```
