# Hamrah onboarding v2 (26 screens + direct-to-chat)

The `FLOW` in `src/data.js` is the authoritative ordered list. Legacy research,
expert and testimonial pages remain routable from the completed experience; they
are not onboarding steps. `src/screens/Onboarding.jsx` uses the existing `Shell`,
`PageHeader`, `ChoiceButton`, `PrimaryButton`, tokens, typefaces and RTL layout.

- Added: identity selection, five-state mood slider with scalable vector scribbles,
  multi-select restorative activities (including no selection), informational
  check-in screen, and acquisition/source question.
- New mood and check-in illustration are SVG authored for Hamrah, not imported
  screenshots. All five mood values are selectable by pointer, touch or keyboard.
- Default first mood is the middle value but becomes a real response only if changed.
- Profile saves only approved fields (`displayName`, `conversationStyle`). Mood,
  beliefs, identity, therapy history, etc. are NOT uploaded or persisted.
- First completed onboarding lands in Chat. An ephemeral greeting reflects the
  user's current-session name, selected mood and up to one concern/goal. The
  greeting is not persisted to DB, sent to the LLM or retained after reload;
  therefore later AI responses are **not** personalized with these inputs yet.
  Persisted personalization requires a separately reviewed consent, retention,
  access-control and prompt-safety design.
- The check-in screen is informational. **No push permission or background
  notification scheduler is implemented here.** Do not claim reminders are live
  until the notification feature is connected.
- Onboarding v2 backend returns `hamrah-v2` and allows ordinals 0..25. Run SQL
  migration `0014_onboarding_v2_cursor.sql` before deploying the matching new
  frontend and API together. Incomplete v1 users will be remapped to an earlier
  safe checkpoint; completed user profiles are unchanged.
- Check 320/360/390/430px widths, short viewports, keyboard slider, progress
  resume/back, multiple-click prevention, Ready recovery and direct-to-chat.
