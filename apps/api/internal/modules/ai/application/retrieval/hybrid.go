package retrieval

import (
	"golang.org/x/text/unicode/norm"
	"math"
	"sort"
	"strings"
	"unicode"
)

// Terms normalizes Arabic/Persian yeh/kaf, diacritics, digits and joiners.
// Joiners are token boundaries so نیم‌فاصله and نیم فاصله share tokens. No
// language-specific intent phrases or guessed stemming affect query meaning.
func Terms(text string) []string {
	text = strings.ToLower(norm.NFKC.String(text))
	var words []string
	var word []rune
	flush := func() {
		if len(word) > 0 {
			words = append(words, string(word))
			word = nil
		}
	}
	for _, r := range text {
		switch {
		case r == 'ي' || r == 'ى':
			r = 'ی'
		case r == 'ك':
			r = 'ک'
		case r >= '۰' && r <= '۹':
			r = '0' + r - '۰'
		case r >= '٠' && r <= '٩':
			r = '0' + r - '٠'
		}
		if unicode.IsMark(r) || r == 'ـ' {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
		} else {
			flush()
		}
	}
	flush()
	return words
}

// Standard lexical stop words remove grammatical noise from BM25, not from
// semantic queries or deduplication. Negations are deliberately retained.
var lexicalStop = func() map[string]bool {
	words := map[string]bool{}
	for _, w := range strings.Fields(`a an the and or of to in on at for from with by as is are was were be been being am do does did have has had i me my we us our you your he him his she her it its they them their this that these those what which who how when where about
 از به با برای در بر را و یا که این آن اون هم یک یه من ما تو شما او آنها ها های می است هست بود بوده بودند هستم هستی هستیم بودم بودی بودیم کنم کنی کنیم کند کردم کردی کردیم کنه درباره چه چی چطور چگونه چرا آیا`) {
		words[w] = true
	}
	return words
}()

func lexicalTerms(text string) []string {
	var out []string
	for _, t := range Terms(text) {
		if !lexicalStop[t] {
			out = append(out, t)
		}
	}
	return out
}

type ranked struct {
	index int
	score float64
}

func order(scores []ranked, docs []Candidate) []ranked {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		a, b := docs[scores[i].index], docs[scores[j].index]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.ID.String() < b.ID.String()
	})
	return scores
}

// BM25 is retained for possible reactivation; active Retrieval does not call it.
// BM25 uses Okapi TF saturation and document-length normalization. Corpus
// statistics use only authorized sources, including those awaiting embeddings.
func bm25(query string, docs []Candidate) []ranked {
	if len(docs) == 0 {
		return nil
	}
	terms := map[string]bool{}
	for _, t := range lexicalTerms(query) {
		terms[t] = true
	}
	queryTerms := make([]string, 0, len(terms))
	for term := range terms {
		queryTerms = append(queryTerms, term)
	}
	sort.Strings(queryTerms)
	tf := make([]map[string]int, len(docs))
	lengths := make([]int, len(docs))
	df := map[string]int{}
	total := 0
	queryHashes := map[string]string{}
	for _, term := range queryTerms {
		queryHashes[term] = TermHash(term)
	}
	for i, d := range docs {
		tf[i] = map[string]int{}
		if d.Lexical != nil {
			lengths[i] = d.Lexical.Length
			total += lengths[i]
			for _, term := range queryTerms {
				if n := d.Lexical.Frequencies[queryHashes[term]]; n > 0 {
					tf[i][term] = n
				}
			}
		} else {
			tokens := lexicalTerms(d.Text)
			lengths[i] = len(tokens)
			total += len(tokens)
			for _, term := range tokens {
				if terms[term] {
					tf[i][term]++
				}
			}
		}
		for term := range tf[i] {
			df[term]++
		}
	}
	if total == 0 {
		return nil
	}
	avg := float64(total) / float64(len(docs))
	const k1, b = 1.2, 0.75
	var out []ranked
	for i := range docs {
		score := 0.
		for _, t := range queryTerms {
			f := tf[i][t]
			if f == 0 {
				continue
			}
			freq := float64(f)
			idf := math.Log(1 + (float64(len(docs)-df[t])+0.5)/(float64(df[t])+0.5))
			score += idf * freq * (k1 + 1) / (freq + k1*(1-b+b*float64(lengths[i])/avg))
		}
		if score > 0 {
			out = append(out, ranked{i, score})
		}
	}
	return order(out, docs)
}
func vectorSearch(query []float32, docs []Candidate) []ranked {
	var out []ranked
	for i, d := range docs {
		if score, ok := cosine(query, d.Vector); ok && score > 0 {
			out = append(out, ranked{i, score})
		}
	}
	return order(out, docs)
}

// RRF combines ranks rather than incomparable branch scores. Each active branch votes
// equally; the conventional k=60 smooths rank differences. No learned/manual weights.
func fuse(docs []Candidate, lists ...[]ranked) []ranked {
	scores := map[int]float64{}
	for _, list := range lists {
		for rank, item := range list {
			scores[item.index] += 1 / (60 + float64(rank+1))
		}
	}
	out := make([]ranked, 0, len(scores))
	for i, score := range scores {
		out = append(out, ranked{i, score})
	}
	return order(out, docs)
}
