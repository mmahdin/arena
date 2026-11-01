# START HERE — The Path-inspired Persian AI Companion MVP

**Language of these instructions: English. Language of the actual app UI: Persian (Farsi), with true right-to-left layout.**

This folder is the **complete English-language design reference handoff** for designing and building the frontend-only interactive MVP. It contains all **61 original reference screenshots**, five overview storyboards, detailed visual analysis, source-inspired tokens, screen-to-feature mappings, a component/state inventory, a ready-to-paste Figma Make build prompt, mock Persian-language content, and an offline visual image browser.

## Start with these three things

1. **`FIGMA_MAKE_MASTER_PROMPT_EN.md`** — Paste this as your main instruction to Figma Make (or your AI design/build tool). This defines **our** product: onboarding, five AI communication styles with same-prompt response comparisons, multiple chat threads, contextual memory controls, a self-understanding profile, and mock assessments.
2. **`VISUAL_REFERENCE_BROWSER_EN.html`** — Open in a browser **after extracting the ZIP**. It lets you browse all 61 screenshots in order, search the design notes, and jump to full-size original images. The content is in English; all images are local.
3. **`VISUAL_REFERENCE_BROWSER_STANDALONE_EN.html`** — A second, self-contained browser with all reference images embedded. It works as a single HTML file without an extracted `screens/` folder (but supporting handoff documents still require the package).
4. **`docs/THE_PATH_61_SCREEN_AUDIT_EN.md`** — Deep per-screen analysis with visible layout, component recommendations, and a specific adopt/adapt/avoid decision for **each of the 61 source screenshots**.

## Recommended Figma reference attachments

**Attach the master prompt and the source reference package to the same project when possible.** Figma Make and related tools may not automatically unzip an archive or read every file inside it. **If a ZIP is not recognized, extract it locally and upload the relevant `.md` text files, storyboards and individual JPGs.** Do not assume an AI designer has inspected files merely because they were attached.

Use `docs/REFERENCE_TO_MVP_MAPPING_EN.md` to select specific screenshots for a feature. Good starting screenshots:

- Welcome: `screens/photo_57_2026-11-01_13-06-40.jpg`, `screens/photo_53_2026-11-01_13-06-40.jpg`
- Questionnaire: `screens/photo_56_2026-11-01_13-06-40.jpg`, `screens/photo_52_2026-11-01_13-06-40.jpg`, `screens/photo_47_2026-11-01_13-06-40.jpg`
- Emotional pauses: `screens/photo_46_2026-11-01_13-06-40.jpg`, `screens/photo_36_2026-11-01_13-06-40.jpg`
- Style-selection visual quality: `screens/photo_20_2026-11-01_13-06-40.jpg`, `screens/photo_11_2026-11-01_13-06-40.jpg`, `screens/photo_9_2026-11-01_13-06-40.jpg`
- Chat: `screens/photo_5_2026-11-01_13-06-40.jpg`, `screens/photo_4_2026-11-01_13-06-40.jpg`
- Home: `screens/photo_22_2026-11-01_13-06-40.jpg`, `screens/photo_3_2026-11-01_13-06-40.jpg`
- Reflective cards: `screens/photo_2_2026-11-01_13-06-40.jpg`

## Folder structure

```text
The_Path_English_Figma_Reference_Package/
  START_HERE.md
  FIGMA_MAKE_MASTER_PROMPT_EN.md
  VISUAL_REFERENCE_BROWSER_EN.html
  VISUAL_REFERENCE_BROWSER_STANDALONE_EN.html
  docs/
    THE_PATH_61_SCREEN_AUDIT_EN.md       <-- all 61, English, with image links
    REFERENCE_TO_MVP_MAPPING_EN.md       <-- explicit source-to-feature decisions
    COMPONENTS_STATES_AND_FLOWS_EN.md    <-- design system + behavior coverage
    ORIGINALITY_SAFETY_AND_LIMITS_EN.md  <-- limitations and non-copy rules
  data/
    screen_manifest_en.json              <-- 61 machine-readable screen records
    reference_tokens_en.json             <-- visual approximations + recommendations
    mock_app_data_fa.json                <-- Persian sample UI data for frontend demo
    source_image_sha256.json             <-- screenshot integrity / coverage
  screens/
    photo_61_2026-11-01_13-06-40.jpg
    ...
    photo_1_2026-11-01_13-06-40.jpg
  storyboards/
    storyboard_01_61-49.jpg
    storyboard_02_48-37.jpg
    storyboard_03_36-25.jpg
    storyboard_04_24-13.jpg
    storyboard_05_12-1.jpg
```

## Golden rules

- **Design our MVP, not a 61-screen clone.** Each screenshot is a reference to borrow principles, not a request to reproduce that app's flow.
- **All interface text is Persian and RTL.** Instructions and rationale are English; the mock app content is deliberately Persian.
- **Use abstract artwork by default for communication styles.** No real-person therapist photos or invented human professional identities.
- **Do not build timed therapy sessions, facial emotion analysis, mandatory camera permissions, referral marketing questionnaires or copied testimonials.** They appear in the reference but are not in our MVP.
- **The current app must be interactive but frontend-only**, driven by replaceable mock data. No real AI or backend integration yet.
- **The screenshot audit separates observed design from proposed interactions** and marks nonverifiable claims.
- **Source screenshots are unchanged** and are for internal visual reference. Do not reuse third-party imagery, logos, statistics, quotes, or names in the released product.

## Expected completion

A working, premium-quality, mobile-first interactive frontend, realistically populated with Persian content, featuring onboarding, five communication style variants, reply comparison, multiple mock conversations, appropriately scoped simulated memory, editable personal observations, an assessment library and one end-to-end illustrative assessment. The result should have reusable components, coherent mobile navigation, responsive 360 / 390 / 430px layouts, and no dead ends in the core journey.
