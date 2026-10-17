package docs

import (
	"errors"
	"sort"
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

// stopwords are the English words that are on almost every line, so a line
// that holds one tells nothing about the query.
var stopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "by": true,
	"can": true, "do": true, "does": true, "for": true, "from": true, "has": true, "have": true,
	"how": true, "in": true, "is": true, "it": true, "its": true, "of": true, "on": true, "or": true,
	"that": true, "the": true, "this": true, "to": true, "was": true, "what": true, "when": true,
	"which": true, "who": true, "why": true, "with": true,
}

// wordsOf returns the lowercase runs of letters and digits in s, as the
// FTS5 unicode61 tokenizer splits them.
func wordsOf(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// stemPrefix cuts a word to a prefix that its inflections share
// ("tolerance", "tolerances", "tolerant"), in place of the porter stemmer
// of the index.
func stemPrefix(w string) string {
	if len(w) <= 5 {
		return w
	}
	return w[:max(5, len(w)-3)]
}

// Line is one numbered line of a document.
type Line struct {
	N    int    `json:"line"`
	Text string `json:"text"`
}

// MatchingLines returns lines of a passage that fit in budget bytes, in line
// order: first the lines that hold the most distinct query words (a stopword
// does not count), then the lines next to them, nearest first, so a line
// that completes a match is shown too. body is the passage text and start
// its first line number. Each line costs its length plus 8 bytes; a blank
// line is never shown. Each line is cut to maxRunes characters first.
func MatchingLines(body string, start int, query string, budget, maxRunes int) []Line {
	var terms []string
	seen := map[string]bool{}
	for _, w := range wordsOf(query) {
		if stopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		terms = append(terms, stemPrefix(w))
	}
	lines := strings.Split(body, "\n")
	text := make([]string, len(lines))
	type scored struct{ i, score int }
	var cands []scored
	for i, line := range lines {
		text[i] = lineWindow(CleanLine(line), terms, maxRunes)
		words := wordsOf(line)
		score := 0
		for _, t := range terms {
			for _, w := range words {
				if strings.HasPrefix(w, t) {
					score++
					break
				}
			}
		}
		if score > 0 {
			cands = append(cands, scored{i, score})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].score > cands[j].score })

	picked := map[int]bool{}
	take := func(i int) bool {
		if i < 0 || i >= len(lines) || picked[i] || text[i] == "" {
			return false
		}
		if cost := len(text[i]) + 8; cost <= budget {
			budget -= cost
			picked[i] = true
			return true
		}
		return false
	}
	for _, c := range cands {
		take(c.i)
	}
	// The lines next to the matches, nearest first.
	for d := 1; d < len(lines) && len(picked) > 0; d++ {
		var from []int
		for i := range picked {
			from = append(from, i)
		}
		sort.Ints(from)
		added := false
		for _, i := range from {
			added = take(i-d) || added
			added = take(i+d) || added
		}
		if !added && budget < 16 {
			break
		}
	}
	var out []Line
	for i := range lines {
		if picked[i] {
			out = append(out, Line{start + i, text[i]})
		}
	}
	return out
}

// WordsNotIn returns the terms of query (as MatchExpression splits them)
// that have a word that title does not hold, joined by spaces. A "quoted
// phrase" stays one term.
func WordsNotIn(query, title string) string {
	have := map[string]bool{}
	for _, w := range wordsOf(title) {
		have[w] = true
	}
	var out []string
	for _, t := range splitQuery(query) {
		for _, w := range wordsOf(t) {
			if !have[w] && !stopwords[w] {
				if strings.ContainsAny(t, " \t") {
					t = `"` + t + `"`
				}
				out = append(out, t)
				break
			}
		}
	}
	return strings.Join(out, " ")
}

// lineWindow cuts a line to maxRunes characters. A longer line that holds
// a query term keeps the part with the most distinct terms (then the most
// matches, then the earliest), so the answer deep in a long line, such as a
// speaker turn of a transcript, is printed and not only the start of the
// line. A cut end is marked with "…". terms are lowercase word prefixes.
func lineWindow(line string, terms []string, maxRunes int) string {
	rs := []rune(line)
	if len(rs) <= maxRunes {
		return line
	}
	isWord := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
	// The first rune of each word that matches a term, and the term.
	type hit struct{ pos, term int }
	var hits []hit
	for i := 0; i < len(rs); {
		if !isWord(rs[i]) {
			i++
			continue
		}
		j := i
		for j < len(rs) && isWord(rs[j]) {
			j++
		}
		w := strings.ToLower(string(rs[i:j]))
		for t, term := range terms {
			if strings.HasPrefix(w, term) {
				hits = append(hits, hit{i, t})
				break
			}
		}
		i = j
	}
	// A part has room for maxRunes-2 runes between its two marks. It starts
	// a quarter of the width before a match, so the words that lead to the
	// match are printed too.
	// The start of the line is the first candidate, so a tie keeps it.
	width, lead := maxRunes-2, maxRunes/4
	starts := []int{0}
	for _, h := range hits {
		starts = append(starts, max(0, h.pos-lead))
	}
	best, bestTerms, bestHits := 0, -1, 0
	for _, from := range starts {
		seen := map[int]bool{}
		n := 0
		for _, o := range hits {
			if o.pos >= from && o.pos < from+width {
				seen[o.term] = true
				n++
			}
		}
		if len(seen) > bestTerms || len(seen) == bestTerms && n > bestHits {
			best, bestTerms, bestHits = from, len(seen), n
		}
	}
	if best == 0 {
		return Truncate(line, maxRunes)
	}
	for best < len(rs) && isWord(rs[best-1]) {
		best++
	}
	end := min(len(rs), best+width)
	out := "…" + strings.TrimSpace(string(rs[best:end]))
	if end < len(rs) {
		out += "…"
	}
	return out
}
