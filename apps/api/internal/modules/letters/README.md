# Saved personal letters

Owner-scoped read and soft-delete APIs preserve existing letters, including after
a source conversation is hidden. No table, payload or source reference is removed.

WP-16 retires the generator's dependency on the experimental AI gateway and phrase
safety. No new personalized letters can be generated until there is an approved
AI Core letter contract and reviewed content. The existing POST route can replay
an existing eligible saved result; attempts requiring generation return 503.
It never calls the prototype or fabricates a replacement. Existing independently
injected application test doubles verify persistence only, not clinical quality.
