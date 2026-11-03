# Chat/navigation transition polish — frontend only

## Bug and fix

Every route changed the keyed `.screen-transition` wrapper. The old `screen-in`
keyframes started at `opacity: 0`, exposing the light-colored `.app-shell` behind
newly mounted dark Chat, History, Home and Menu pages. That caused an abrupt pale
flash whenever navigating out of Chat.

The shell now uses the **destination route's existing designer surface**
(light, beige, dark or black), and the wrapper shares this surface. The incoming
screen starts at 96% opacity with a 4px offset rather than disappearing completely.
Animation is 200ms and disabled under `prefers-reduced-motion: reduce`.

No product screen, button, chat state, API, route permissions, image or font changed.
No backend change or migration is necessary. The existing UI palette is unchanged.

## Checks

- `npm test` and `npm run check:foundation` for logic and source regressions.
- `npm run build` after dependencies are available.
- `BROWSER_BIN=/usr/bin/chromium npm run test:e2e:transitions` with Vite, Go/MSSQL
  and Playwright running locally. It checks Chat → Home → Menu → History at mobile
  widths 360, 390 and 430, matching dark backdrops and reduced-motion behavior.
- Manually verify navigation to/from Chat, especially immediately after onboarding,
  with a screenshot or video on the target device.

The browser check does not mock backend responses or claim to verify screen pixel identity.
