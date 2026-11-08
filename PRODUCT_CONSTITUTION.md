# Product Constitution — کنار / Kenar

## The designer UI is the product-intent authority
The current uploaded JSX/CSS implementation establishes the **کنار (Kenar)** brand, its stone-and-shore illustrations, mobile-first tone, 26-step onboarding sequence, home/letters, history, chat and message actions. Keep these screens and their layout, navigation and interaction system, including choices that await separate backend work. An agent must not discard them simply because the former technical MVP is smaller.

## Brand identity is now Kenar, and it is original
The floral, Ash-inspired artwork has been replaced by the Kenar identity documented in `docs/design/KENAR_VISUAL_IDENTITY.md`: two river stones resting on one shore line, the "patient stone" companion, and a paper/sand/ochre/pine palette. The mark, the companion, the shore and the palette must stay coherent. Do not reintroduce flower, petal, stem or daisy imagery, and do not swap the companion for an arbitrary decorative symbol: the identity carries the meaning of the name (کنار — to be beside someone). Changing the artwork is a legitimate brand task; changing layout geometry, navigation, spacing or control behaviour is not.

## Distinguish design authority from verified claims
Visual design cannot prove statements about medical effectiveness, institutional research affiliations, expert credentials, 6000+ reviews, therapy, data location, account deletion or voice/analysis functionality. The current UI has such claims. Before a screen is exposed in the connected tester flow: substantiate or replace factual copy while keeping its design geometry. The product is an AI tool, not a human clinician; do not imply diagnosis, prescription, medical service, guaranteed outcomes or active crisis monitoring.

## Sensitive question and privacy boundary
The onboarding asks therapy history, neurodivergence, faith, age, relationships and concerns. These are *design choices requiring a separate data-minimization, consent, safety and backend decision*. Until approved, do not persist or transmit any answers, even if the screen remains previewable. The current Backend profile accepts only display name and conversation style. The current `shareAnonymized` UI toggle does not implement any actual sharing policy; defaulting it ON is not proof of consent. Fix/gate such flows before a connected user can rely on them.

## Release boundaries
Internal, private, synthetic tester data only. Signed revocable Dev Session is an identity selector, not authentication. Avoid real sensitive health data, analytics, external SDKs, voice/video collection, clinical metrics, real-person testing or open hosting. Backend keyword safety is not clinically validated. Claims about 'securely stored on phone' are not compatible with MSSQL server-side chat storage.

## Interaction quality
Preserve user control, accessible RTL, meaningful error states, disabled fake actions, explicit destructive confirmation and continued conversation after reload. Avoid dark patterns: a complete onboarding design does not justify forcing unneeded sensitive questions before chat. Let users navigate back/skip when a question is not necessary for the approved feature.
