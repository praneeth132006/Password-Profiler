package mutate

import (
	"strings"
	"unicode"
)

// CaseFunc maps a word to a single re-cased form.
type CaseFunc func(string) string

// caseFuncs maps config names to their transform. identity is included so a
// token can pass through untouched alongside its recased siblings.
var caseFuncs = map[string]CaseFunc{
	"identity":   Identity,
	"lower":      Lower,
	"upper":      Upper,
	"capitalize": Capitalize,
	"toggle":     Toggle,
	"invert":     Invert,
}

// Identity returns the word unchanged.
func Identity(s string) string { return s }

// Lower lowercases the whole word.
func Lower(s string) string { return strings.ToLower(s) }

// Upper uppercases the whole word.
func Upper(s string) string { return strings.ToUpper(s) }

// Capitalize uppercases the first rune and lowercases the rest ("john" -> "John").
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	first := unicode.ToUpper(r[0])
	rest := strings.ToLower(string(r[1:]))
	return string(first) + rest
}

// Toggle uppercases the first rune, leaves the rest as-is ("john" -> "John",
// "jOHN" -> "JOHN"). Distinct from Capitalize, which forces the tail lower.
func Toggle(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Invert swaps the case of every rune ("John" -> "jOHN").
func Invert(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsUpper(r):
			return unicode.ToLower(r)
		case unicode.IsLower(r):
			return unicode.ToUpper(r)
		default:
			return r
		}
	}, s)
}
