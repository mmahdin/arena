package retrieval

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// LexicalDocument is a rebuildable search projection, not a new user memory.
// Token hashes/counts are private index metadata and never model context.
// Frequencies may contain only the current query's keys after SQL projection;
// Length and Signature still describe the complete authorized document.
type LexicalDocument struct {
	Frequencies map[string]int
	Length      int
	Identity    string
	Signature   []byte
}

func TermHash(term string) string { v := sha256.Sum256([]byte(term)); return hex.EncodeToString(v[:]) }
func BuildLexical(text string) *LexicalDocument {
	out := &LexicalDocument{Frequencies: map[string]int{}, Signature: make([]byte, 32)}
	for _, term := range lexicalTerms(text) {
		out.Length++
		h := TermHash(term)
		out.Frequencies[h]++
		raw, _ := hex.DecodeString(h)
		bucket := int(raw[0])
		out.Signature[bucket/8] |= 1 << uint(bucket%8)
	}
	out.Identity = TermHash(strings.Join(Terms(text), " "))
	return out
}
func QueryHashes(query string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, term := range lexicalTerms(query) {
		hash := TermHash(term)
		if !seen[hash] {
			seen[hash] = true
			out = append(out, hash)
		}
	}
	return out
}
