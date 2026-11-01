# Source Evidence, Originality, Privacy and Design Limitations

## 1. This is a visual reference audit, not product reverse engineering

The 61 screenshots are user-provided visual evidence of interface states. They do not prove The Path's exact underlying source code, components, back-end AI capabilities, accessibility compliance, behavioral data model, clinical validity or user outcomes. Names of our proposed components are **recommendations**, never an assertion that The Path's source uses those names.

The images may include Android status indicators, an open keyboard, scroll offsets, temporary clipboard banners and screenshot detection notices. These are not necessarily part of the normal page design. Specifically, a temporary toast on `photo_25` or `photo_4` should not be implemented as a permanent UI element.

## 2. Avoid copying protected identity assets

- Never ship The Path's exact brand mark, graphics, named personalities, photographed faces, on-screen excerpts, testimonials or analytics percentages as our own.
- A visual similarity goal is better expressed as **design principles:** visual cadence, editorial hierarchy, warm restrained palette, atmospheric gradients, curved home header, pill buttons, minimal controls and calm layouts.
- Recreate our own abstract communication-style artwork and distinctive original brand identity. Avoid real-person-therapist photos by default.
- Source JPGs are preserved in this folder for private reference / internal design analysis only.

## 3. Clinical and personal-data boundaries

- A psychological companion is not a licensed human clinician. Do not imply this product has treatment efficacy, diagnosis ability or crisis intervention capability when those haven't been developed and validated.
- Do not borrow clinical questionnaire wording from screenshots as a finished validated assessment. Test rights, cultural fit, scoring and safety separately.
- Mood/reflection mock text is illustrative; it must not be represented as AI inference from real user data.
- Cross-chat memory should be contextual, minimal, understandable and user-editable. A frontend mock memory store is not a secure clinical database.
- The precise jurisdiction for users and infrastructure is not finalized. A US number such as 988 in reference `photo_58` must never be presented as universal emergency coverage.

## 4. Contrast and accessibility observations

Source photos contain examples of tiny muted text, light gray disabled states, white text on bright gradients and lengthy paragraphs on small screens. These should trigger **design tests**, not blind replication. Test real Persian text wrapping, font scaling, 44px touch targets, focus visibility, reduced-motion alternatives, keyboard overlap and TalkBack/VoiceOver semantics where possible.

## 5. Assumptions of the new MVP

- The app language is Persian and the layout is RTL.
- The frontend is mobile-first and interactive, with mock data and simulated AI replies.
- No live inference engine, API, database, privacy encryption or real user analytics are in scope for this deliverable.
- The five proposed communication styles are **interaction patterns**, not therapy professionals.
- Onboarding questions and psychological assessments are placeholders pending appropriate research.
- Assessment results are non-diagnostic descriptions; user must control whether they enter memory/profile.
- Voice, video, facial expressions, session timers, clinical notes, therapist booking, payments and referrals are out of scope now.

## 6. Safely applying a visual reference

Start from the user's new product requirements. Choose reference screens by feature using `REFERENCE_TO_MVP_MAPPING_EN.md`. Use source images to inspect color, typography, spacing, proportions and emotion. Translate into an original token set and component design. Verify the resulting app **without** requiring user familiarity with The Path. The 61 source screenshots are an archive of influences, not a 61-step flow to reproduce.
