package keyword

import (
	"strings"
	"unicode"
)

// Match is a single keyword occurrence found inside a transcription.
type Match struct {
	// Keyword is the keyword exactly as written in the configuration.
	Keyword string
	// Phrase is the transcription the keyword was found in.
	Phrase string
}

type entry struct {
	original string
	tokens   []string
}

// Matcher looks for configured keywords in transcriptions.
//
// A keyword matches only on whole words, so "aiuto" does not fire on "aiutone"
// or "inaiuto". Keywords containing spaces are matched as sequences of
// consecutive words.
type Matcher struct {
	// byLength groups the keywords by word count, so a single pass over the
	// tokens is enough and longer keywords are tested first.
	byLength map[int][]entry
	maxLen   int
}

// New builds a Matcher for the given keywords. Keywords are normalized the same
// way transcriptions are, so matching is insensitive to case and punctuation.
func New(keywords []string) *Matcher {
	m := &Matcher{byLength: make(map[int][]entry)}
	for _, k := range keywords {
		tokens := tokenize(k)
		if len(tokens) == 0 {
			continue
		}
		n := len(tokens)
		m.byLength[n] = append(m.byLength[n], entry{original: k, tokens: tokens})
		if n > m.maxLen {
			m.maxLen = n
		}
	}
	return m
}

// Empty reports whether the matcher has no usable keyword.
func (m *Matcher) Empty() bool { return m.maxLen == 0 }

// Find returns every keyword occurrence in phrase, in order of appearance.
func (m *Matcher) Find(phrase string) []Match {
	if m.Empty() {
		return nil
	}

	tokens := tokenize(phrase)
	var matches []Match

	for i := 0; i < len(tokens); i++ {
		max := m.maxLen
		if i+max > len(tokens) {
			max = len(tokens) - i
		}
		for n := max; n >= 1; n-- {
			window := tokens[i : i+n]
			for _, e := range m.byLength[n] {
				if equalTokens(window, e.tokens) {
					matches = append(matches, Match{Keyword: e.original, Phrase: phrase})
				}
			}
		}
	}
	return matches
}

// tokenize lowercases the text, turns every non alphanumeric character into a
// space and splits the result into words.
func tokenize(s string) []string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

func equalTokens(window, keyword []string) bool {
	for i := range keyword {
		if window[i] != keyword[i] {
			return false
		}
	}
	return true
}
