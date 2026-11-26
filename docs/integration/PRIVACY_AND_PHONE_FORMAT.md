# Privacy setting & phone entry — UI-safe incremental change

- The existing `privacy` route and switch remain in the original Onboarding.jsx layout. Default server preference is ON in the current **internal synthetic-only** environment; a save updates only the session user's setting in MSSQL. The setting does not transmit chat content.
- Privacy state is loaded on entering the route, with error/retry and a session-generation fence against late responses after switching users. The switch is disabled until the server preference loads. No localStorage/cookie value is read or saved by JS.
- Phone-shaped entry accepts 11 digits beginning `09` (Persian/Arabic digits normalized), with no narrow `0900000` restriction. The entry screen omits technical disclaimers. **This is not number ownership verification** and local test identifiers must not be personal phone numbers.
- Existing designer screens, assets and buttons are left unchanged apart from the switch wiring, input hint, and removal of two technical sign-in paragraphs.
- No `index.staff.html` file exists in this frontend snapshot; the shared React `privacy` route implementation is updated. If a separately customized staff entrypoint exists in another checkout, its routing must be verified locally.
- Unit: `npm test`, `npm run check:foundation`. Build: `npm ci && npm run build`. Live: test reload/toggle/two users with Vite proxy → Go/MSSQL and all previous E2E suites.
