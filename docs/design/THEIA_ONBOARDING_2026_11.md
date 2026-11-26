# Theia Onboarding — Earthshine (2026-11)

## Core Principle

**One subject. One variable. One warm light.**

Across onboarding, the user stays inside a single nocturnal sky before sunrise:

1. **Subject** — The Moon, always present in the upper-central sky as a whole astronomical sphere: a waning morning crescent (`cool white`) plus its earthlit dark hemisphere (`desaturated blue-gray` with faint lunar maria).
2. **Variable** — How readable the unlit portion of the Moon becomes through **earthshine** as the eye adjusts (`~8%` on Welcome → `~20%` Night → `~50%` Seeing → `~70%` Dawn → `100%` Three Lights, Check-ins, and Final Chapter).
3. **Warmth** — Rationed muted amber (`#C99867`) withheld until the horizon enters in Scene 5 (false dawn) and sunrise approaches in Scene 6.

Central philosophy: **Revelation rather than Illumination** — Theia adds nothing artificial to the user; she helps what is already present become visible. Theia is never depicted as a person, mascot, or character.

## Visual & Motion Discipline

- **Matte tonal fields + fine film grain** (`src/components/Earthshine.jsx`, `src/onboarding.css`), generous negative space, no outlines, no arbitrary rings, no 4-point sparkle stars, no lens flares, and no cultural/architectural props.
- **Astronomical honesty** — Two-arc geometric crescent with a multi-step soft penumbra along the terminator and an always-present ashen disc so the Moon reads as a real celestial body, never a flat crescent or crescent-and-star emblem.
- **Horizon** — Appears for the first time in Scene 5 (`know-intro` / سپیده) as a thin line; the sea below is implied solely by darker tonal value.
- **Reversed Finale** — Scene 8 (`ready` / آغاز مسیر) uses the exact composition and lunar scale of Scene 1 (`welcome`), with tonal values reversed into pale civil twilight (`#BCC7D4`–`#E7D5C4`).
- **Motion** — Slow opacity cross-fades, gradual star appearance, 18s imperceptible false-dawn breathing, and a 1.5s slow fade on Venus when reminders are switched off. All animations and transitions yield to `prefers-reduced-motion: reduce`.

## Storyboard Mapping

| # | Route | Chapter | Scene Key | Visual Progression |
|---|---|---|---|---|
| 1 | `welcome` | آغاز | `welcome` | Near-black sky, thin crescent + faint glare halo, barely-there ashen disc |
| 2 | `privacy` | آغاز | `privacy` | Flat dark shutter plane with crisp vertical opening + symbolic aperture drag/slider; separate backend consent switch + plain AI companion disclosure |
| 3 | `simple-intro` | شب | `night` | Full ultramarine sky, 7 quiet stars, ~20% earthshine, zero warm light |
| 4 | `deeper-intro` | دیدن | `seeing` | Intimate close-up Moon, ~50% earthshine, visible lunar maria, relaxed halo |
| 5 | `know-intro` | سپیده | `dawn` | Thin horizon enters, pale rose-gray band + faint vertical false-dawn cone, ~70% earthshine |
| 6 | `almost-intro` | سه نور | `threeLights` | Widest morning sky, bright narrow horizon band with first solar rim at one point, 100% earthshine |
| 7 | `checkins` | سه نور | `checkins` | Pale morning sky, 100% earthshine, steady warm-white Venus offset naturally; slow fade when reminders are toggled off |
| 8 | `ready` | آغاز مسیر | `final` | Scene 1 framing with reversed civil-twilight values, quiet 100% earthshine, thin warm horizon, fading Venus |

## Preserved Contracts

- All 26 `FLOW` routes, ordinals, intake questions (`QUESTIONS`), profile save (`PUT /api/v1/profile`), onboarding cursor (`GET/PUT /api/v1/onboarding-progress`), and privacy switch (`GET/PATCH /api/v1/privacy-preferences`) remain unchanged.
- On `privacy`, the vertical aperture responds symbolically to touch/slider (`aria-label="تنظیم نمادین گشودگی دریچه در تصویر"`) and is strictly decoupled from `onTogglePrivacy`.
