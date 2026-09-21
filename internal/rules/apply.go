package rules

import (
	"fmt"
	"strings"
	"unicode"
)

// Apply runs a single hashcat rule line against word and returns the result.
// It supports the subset of the rule language that Generate emits, which is
// enough to validate generated rules without invoking hashcat:
//
//	:        no-op            l  lowercase all      u  uppercase all
//	c        capitalize       C  invert-capitalize  t  toggle-case all
//	TN       toggle at pos N  r  reverse            d  duplicate
//	]        delete last      [  delete first
//	$X       append X         ^X prepend X          sXY substitute all X->Y
//
// An unknown function returns an error, so a malformed rule is never silently
// treated as a no-op.
func Apply(word, rule string) (string, error) {
	r := []rune(rule)
	w := word
	for i := 0; i < len(r); {
		fn := r[i]
		switch fn {
		case ' ', '\t':
			i++
		case ':':
			i++
		case 'l':
			w = strings.ToLower(w)
			i++
		case 'u':
			w = strings.ToUpper(w)
			i++
		case 'c':
			w = capitalize(w)
			i++
		case 'C':
			w = invertCapitalize(w)
			i++
		case 't':
			w = toggleAll(w)
			i++
		case 'T':
			pos, err := argPos(r, i+1)
			if err != nil {
				return "", err
			}
			w = togglePos(w, pos)
			i += 2
		case 'r':
			w = reverse(w)
			i++
		case 'd':
			w = w + w
			i++
		case ']':
			w = deleteLast(w)
			i++
		case '[':
			w = deleteFirst(w)
			i++
		case '$':
			if i+1 >= len(r) {
				return "", fmt.Errorf("rule %q: $ missing argument", rule)
			}
			w = w + string(r[i+1])
			i += 2
		case '^':
			if i+1 >= len(r) {
				return "", fmt.Errorf("rule %q: ^ missing argument", rule)
			}
			w = string(r[i+1]) + w
			i += 2
		case 's':
			if i+2 >= len(r) {
				return "", fmt.Errorf("rule %q: s missing arguments", rule)
			}
			w = strings.ReplaceAll(w, string(r[i+1]), string(r[i+2]))
			i += 3
		default:
			return "", fmt.Errorf("rule %q: unsupported function %q", rule, string(fn))
		}
	}
	return w, nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return string(unicode.ToUpper(r[0])) + strings.ToLower(string(r[1:]))
}

func invertCapitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return string(unicode.ToLower(r[0])) + strings.ToUpper(string(r[1:]))
}

func toggleAll(s string) string {
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

func togglePos(s string, pos int) string {
	r := []rune(s)
	if pos < 0 || pos >= len(r) {
		return s
	}
	switch {
	case unicode.IsUpper(r[pos]):
		r[pos] = unicode.ToLower(r[pos])
	case unicode.IsLower(r[pos]):
		r[pos] = unicode.ToUpper(r[pos])
	}
	return string(r)
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func deleteLast(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

func deleteFirst(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[1:])
}

// argPos decodes a hashcat position character (0-9, A-Z) into an index.
func argPos(r []rune, i int) (int, error) {
	if i >= len(r) {
		return 0, fmt.Errorf("position argument missing")
	}
	c := r[i]
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0'), nil
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10, nil
	default:
		return 0, fmt.Errorf("invalid position character %q", string(c))
	}
}
