# Approach screen — minimal swipe selection (2026-11-06)

The screen lets someone choose how Hamrah talks to them by reading one short conversation example. The user explicitly requested few controls and removal of unnecessary text.

## Design basis: ui-ux-pro-max

The installed skill was read and its search tools were run:

```sh
python3 /home/mahdi/.agents/skills/ui-ux-pro-max/scripts/search.py "mental wellness minimal calm" --design-system --variance 1 --density 2
python3 /home/mahdi/.agents/skills/ui-ux-pro-max/scripts/search.py "Minimalism" --domain style --rtl=full -n 1
python3 /home/mahdi/.agents/skills/ui-ux-pro-max/scripts/search.py "select controlled value" --stack react -n 2
```

The verified style is `minimalism-and-swiss-style` with `rtl_level: full`: essential elements, generous whitespace, clear typography, no decorative shadows or gradients, keyboard support and reduced motion. The existing Hamrah neutral tokens, local Persian fonts and floral mark supply the brand treatment. The generated landing-page sections and Latin font pairing are not applicable to this Persian onboarding screen.

The current style appears as plain text. Two subtle unboxed chevrons flanking the name communicate horizontal navigation visually. There is no written gesture instruction, dropdown or row of style choices. The chevrons also work as 44px pointer controls; they become disabled at the respective first/last style. Horizontal mouse/touch dragging changes the style and authored reply; the region supports Left/Right and Home/End on the keyboard.

## Visible content

- One heading: «چطور باهات حرف بزنم؟».
- One plain current-style name between two unboxed directional chevrons.
- One shared example prompt and one visible authored reply.
- One primary «ادامه» action plus quiet «رد کردن»; profile edit uses Save/Cancel.
- One Back button in the existing header position.

The only controls are Back, Continue, Skip/Cancel and the two unobtrusive chevrons. Five small, noninteractive dots below the name indicate the selected position. They follow the physical slide offsets from left to right, independently of Persian text direction; the active mark is darker and elongated, and updates with swipe, arrow buttons and keyboard. The live region also announces the position. No select, visible instructions, descriptions, repeated style names, counters, badges, subtitle or footer note. Examples are static and never call the chat API. A polite live region announces style changes. Hidden/inert replies share a grid slot so selections cannot move the surrounding layout.

## Preserved behavior

The five original approach keys and profile mapping, initial selection, onboarding cursor, back/skip navigation, default-choice recording and profile save/cancel are unchanged. No backend, route, schema, voice or conversation behavior was added. Existing unrelated workspace changes are preserved.

## Validation

- `npm test`: PASS.
- `npm run build`: PASS.
- `BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:approach`: PASS, isolated real-component browser regression. Covers the plain style name, unboxed chevrons with 44px hit targets and correct disabled states, absence of instructions/dropdown, all styles, stable geometry and RTL keyboard, mouse/touch swipe, cancellation, reduced motion, long text, short screens and onboarding/edit callbacks at 360/390/430/1280 widths.
- `npm run check:foundation`: PASS, 41/41. The check now validates the imported `FLOW` array instead of counting source-formatting lines; the onboarding flow itself is unchanged.
- Original-byte baseline: NOT RUN because this is an intentional visual update.
- Connected Go/MSSQL suites: NOT RUN; browser preview is isolated and does not prove backend persistence. The five-tone and profile preference tests now select through the actual keyboard controls instead of querying removed style buttons.

## Local preview

```sh
node scripts/approach-preview.mjs --host 127.0.0.1 --port 5176
```

The fixture renders the real screen with synthetic local callbacks and no server writes. `?edit=1` exercises profile-edit actions.
