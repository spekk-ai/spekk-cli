package docs

import (
	"errors"
	"strings"
	"unicode"
)

// IdentifierTerms returns the lowercased parts of each identifier in text that
// has an internal case change: `parseAllSpecs` gives "parse all specs" and
// `HTTPServer` gives "http server". The index writes them to the terms
// column, so a search for the parts finds the identifier. The default FTS5
// tokenizer already splits `snake_case`, `dotted.name`, and `kebab-case`, and
// it keeps a camelCase identifier as one token, so only the case change needs
// this help.
func IdentifierTerms(text string) string {
	var out []string
	word := []rune{}
	flush := func() {
		if parts := splitCamel(word); len(parts) > 1 {
			out = append(out, strings.Join(parts, " "))
		}
		word = word[:0]
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
			continue
		}
		flush()
	}
	flush()
	return strings.Join(out, " ")
}

// splitCamel splits a word at each lower-to-upper change and before the last
// capital of a run of capitals that a lowercase letter follows (HTTPServer ->
// HTTP Server). Digits stay with the letters before them.
func splitCamel(word []rune) []string {
	if len(word) < 2 {
		return nil
	}
	var parts []string
	start := 0
	for i := 1; i < len(word); i++ {
		prev, cur := word[i-1], word[i]
		boundary := unicode.IsUpper(cur) && (unicode.IsLower(prev) || unicode.IsDigit(prev))
		if !boundary && unicode.IsUpper(prev) && unicode.IsUpper(cur) && i+1 < len(word) && unicode.IsLower(word[i+1]) {
			boundary = true
		}
		if boundary {
			parts = append(parts, strings.ToLower(string(word[start:i])))
			start = i
		}
	}
	parts = append(parts, strings.ToLower(string(word[start:])))
	return parts
}

// ErrEmptyQuery is returned when a search query has no word to match.
var ErrEmptyQuery = errors.New("search: the query has no words")

// MatchExpression turns the text an agent types into a safe FTS5 MATCH
// expression. Each whitespace-separated term, or each "quoted phrase", is
// written as an FTS5 string, so no user text is read as FTS5 syntax (NOT,
// AND, *, :, parentheses, and - are plain characters). The terms are joined
// with OR, and BM25 ranks a passage that matches more of them first.
func MatchExpression(query string) (string, error) {
	var words []string
	for _, t := range splitQuery(query) {
		if strings.ContainsFunc(t, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
			words = append(words, t)
		}
	}
	if len(words) == 0 {
		return "", ErrEmptyQuery
	}
	quote := func(t string) string { return `"` + strings.ReplaceAll(t, `"`, `""`) + `"` }
	var terms []string
	for _, w := range words {
		terms = append(terms, quote(w))
	}
	// Each pair of adjacent words is also a phrase. A rare pair ("pep 508",
	// "kinds of") has a high IDF, so a passage that holds the words side by
	// side ranks above one that holds them apart.
	for i := 0; i+1 < len(words); i++ {
		terms = append(terms, quote(words[i]+" "+words[i+1]))
	}
	return strings.Join(terms, " OR "), nil
}

// splitQuery splits on whitespace and keeps a double-quoted part as one term.
func splitQuery(q string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range q {
		switch {
		case r == '"':
			if inQuote {
				out = append(out, cur.String())
				cur.Reset()
			} else if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
