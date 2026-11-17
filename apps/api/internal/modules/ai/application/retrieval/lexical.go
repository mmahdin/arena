package retrieval

import (
	"sort"
	"strings"
	"unicode"
)

// This file contains engineering syntax only: pronoun/reference markers,
// person-role words and stop words used to build a bounded lexical query. It is
// NOT clinical content, a risk lexicon or a psychological decision policy, and
// it is never used to infer the user's state. Any clinical vocabulary stays in
// the reviewed Knowledge Bundle (WP-03).
//
// Every list here is intentionally small and closed so that behaviour is
// reproducible in tests and review.

// anaphoraMarkers are reference phrases that point at something said earlier.
// When one is present without an explicit entity, the anaphora strategy runs and
// an unresolved referent is reported as ambiguous instead of being guessed.
var anaphoraMarkers = []string{
	"that thing", "that stuff", "the thing i told", "as i said", "like before",
	"same feeling", "same thing", "again", "that time", "back then", "you know what i mean",
	"what i mentioned", "the other thing", "last time i", "that situation",
	"that day", "that night", "the same", "similar to", "that feeling",
}

// personRoles are relationship nouns. They are matched against stored
// memory_item entity roles (EntityRef.Role) and never persisted as new facts.
// Names are not extracted as entities: v1.2 leaves person alias matching
// [OPEN], so a capitalized token is never assumed to be a person.
var personRoles = []string{
	"sister", "brother", "mother", "mom", "mum", "father", "dad", "partner",
	"husband", "wife", "girlfriend", "boyfriend", "friend", "boss", "manager",
	"colleague", "coworker", "co-worker", "roommate", "son", "daughter",
	"grandmother", "grandfather", "aunt", "uncle", "cousin", "therapist",
	"teacher", "classmate", "neighbour", "neighbor",
}

// emotionWords is a small closed label set used only to boost anaphora recall
// inside the configured recent window. It is not a risk detector: safety
// classification belongs exclusively to WP-06.
var emotionWords = map[string]string{
	"anxious": "anxiety", "anxiety": "anxiety", "worried": "anxiety", "worry": "anxiety",
	"nervous": "anxiety", "panic": "anxiety", "panicking": "anxiety",
	"sad": "sadness", "sadness": "sadness", "down": "sadness", "low": "sadness",
	"hopeless": "sadness", "lonely": "loneliness", "alone": "loneliness",
	"angry": "anger", "anger": "anger", "furious": "anger", "irritated": "anger",
	"frustrated": "frustration", "frustrating": "frustration",
	"ashamed": "shame", "shame": "shame", "embarrassed": "shame",
	"guilty": "guilt", "guilt": "guilt", "exhausted": "fatigue", "tired": "fatigue",
	"overwhelmed": "overwhelm", "stressed": "stress", "stress": "stress",
	"worthless": "self_worth", "failure": "self_worth", "confident": "confidence",
	"happy": "happiness", "relieved": "relief", "calm": "calm",
}

// stopWords are grammar words removed from the lexical query. Adjective forms
// are kept so that "worried about my sister" and "worried about sister" share
// one query token, which is the only stemming implemented.
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "you": true, "your": true, "yours": true,
	"with": true, "about": true, "that": true, "this": true, "what": true, "when": true,
	"did": true, "does": true, "was": true, "were": true, "have": true, "has": true,
	"had": true, "are": true, "will": true, "would": true, "could": true,
	"should": true, "can": true, "may": true, "might": true, "must": true, "from": true,
	"like": true, "just": true, "really": true, "very": true, "much": true, "some": true,
	"there": true, "their": true, "them": true, "they": true, "she": true, "her": true,
	"his": true, "him": true, "it's": true, "im": true, "i'm": true,
	"me": true, "my": true, "we": true, "our": true, "us": true, "not": true,
	"don't": true, "dont": true, "is": true, "of": true, "to": true, "in": true,
	"on": true, "at": true, "as": true, "but": true, "if": true, "so": true,
	"then": true, "than": true, "too": true, "also": true, "all": true, "any": true,
	"know": true, "think": true, "feel": true, "feeling": true, "felt": true,
	"tell": true, "told": true, "say": true, "said": true, "saying": true,
	"been": true, "being": true, "get": true, "got": true, "go": true, "going": true,
	"one": true, "two": true, "thing": true, "things": true, "stuff": true,
	"again": true, "before": true, "last": true, "time": true, "day": true,
	"today": true, "yesterday": true, "week": true, "month": true, "year": true,
}

// lexicalQuery is the bounded deterministic query built from the user message.
type lexicalQuery struct {
	Tokens     []string // keyword tokens, order preserved, deduplicated
	Emotion    string   // first recognized emotion label, if any
	Roles      []string // relationship nouns found in the text
	Anaphora   bool     // a reference phrase is present
	RecallVerb bool     // "what did I tell you..."
}

// buildLexicalQuery extracts the cheap signals used by the retrieval router.
// It is deterministic and has no model dependency, so its behaviour can be
// asserted directly in tests.
func buildLexicalQuery(text string) lexicalQuery {
	var q lexicalQuery
	low := strings.ToLower(text)
	for _, marker := range anaphoraMarkers {
		if len(marker) > 3 && strings.Contains(low, marker) {
			q.Anaphora = true
			break
		}
	}
	q.RecallVerb = hasRecallVerb(low)
	fields := strings.FieldsFunc(low, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '-'
	})
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.Trim(f, "'-")
		if f == "" || seen[f] {
			continue
		}
		if label, ok := emotionWords[f]; ok {
			if q.Emotion == "" {
				q.Emotion = label
			}
			// Emotion words stay in the query: an anaphora boost must not
			// replace lexical matching, it only augments ranking.
		}
		for _, role := range personRoles {
			if f == role {
				// Only the literal role word is recorded. Aliases and name
				// matching remain [OPEN] in v1.2, so "mom" is not silently
				// treated as "mother" and no capitalized token becomes a person.
				q.Roles = append(q.Roles, role)
				seen[f] = true
			}
		}
		// Short tokens and stop words never reach the lexical index.
		if len([]rune(f)) < 3 || stopWords[f] {
			continue
		}
		seen[f] = true
		q.Tokens = append(q.Tokens, f)
		if len(q.Tokens) == 12 {
			break
		}
	}
	sort.Strings(q.Roles)
	q.Roles = dedupeSorted(q.Roles)
	return q
}

// overlappingTerms returns the stored-text terms that also appear in the query.
// Matching is exact-token plus one-way prefix ("anxious"/"anxiety", "mother"/
// "mom" are distinct on purpose: no synonym guessing outside the bundle).
func overlappingTerms(text string, tokens []string) []string {
	low := strings.ToLower(text)
	fields := strings.FieldsFunc(low, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '-'
	})
	present := map[string]bool{}
	for _, f := range fields {
		present[strings.Trim(f, "'-")] = true
	}
	var hits []string
	for _, tok := range tokens {
		if present[tok] {
			hits = append(hits, tok)
			continue
		}
		for word := range present {
			if len(tok) >= 4 && (strings.HasPrefix(word, tok) || strings.HasPrefix(tok, word) && len(word) >= 4) {
				hits = append(hits, tok)
				break
			}
		}
	}
	return hits
}

func appendUnique(in []string, v string) []string {
	for _, item := range in {
		if item == v {
			return in
		}
	}
	return append(in, v)
}

func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := in[:1]
	for _, v := range in[1:] {
		if v != out[len(out)-1] {
			out = append(out, v)
		}
	}
	return out
}
