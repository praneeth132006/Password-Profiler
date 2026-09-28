// Package rank orders candidate passwords by their estimated real-world
// likelihood, so the most probable guesses come first. This is what makes a
// long wordlist *accurate*: a cracker that tries the top of a ranked list hits
// far sooner than one grinding an arbitrarily-ordered list.
//
// Scoring is a transparent, deterministic heuristic derived from well-known
// public studies of leaked-password composition (used here defensively, for
// authorized auditing). It reads only the candidate string, so it can rank
// output from this tool or any existing wordlist without extra metadata. It is
// an ordering signal, NOT a password-strength meter.
//
// The dominant human pattern is a capitalized dictionary-ish word, then a short
// run of digits (a year or "123"), then at most one trailing symbol
// ("Falcons2024!"). Candidates matching that shape score highest; leetspeak,
// interior symbols, all-caps, case toggling and duplicated/very-short strings
// score lower because real users pick them less often.
package rank

import (
	"sort"
	"unicode"
)

// Score returns a relative likelihood score for pw; higher means more likely to
// be a human-chosen password. Scores are comparable only to each other.
func Score(pw string) int {
	r := []rune(pw)
	n := len(r)
	if n == 0 {
		return 0
	}

	// Split the tail into trailing symbols, then trailing digits; the remainder
	// is the "core" (the word part in the canonical shape).
	end := n
	trailingSymbols := 0
	for end > 0 && isSymbol(r[end-1]) {
		trailingSymbols++
		end--
	}
	trailingDigits := 0
	for end > 0 && unicode.IsDigit(r[end-1]) {
		trailingDigits++
		end--
	}
	core := r[:end]

	score := 0

	// Length: 8–12 is the sweet spot for policy-constrained human passwords.
	switch {
	case n >= 8 && n <= 12:
		score += 30
	case n >= 6 && n <= 16:
		score += 18
	case n >= 4 && n <= 20:
		score += 8
	}
	if n < 6 {
		score -= 15
	}

	// Core shape: a real word-like core (mostly letters) is the common case.
	coreLetters, coreDigits, coreSymbols := classify(core)
	if len(core) >= 3 && coreLetters == len(core) {
		score += 25 // clean alphabetic core
	} else if coreLetters >= len(core)-coreDigits && coreLetters > 0 {
		score += 8 // mostly letters
	}
	// Digits or symbols embedded INSIDE the core signal leetspeak / awkward
	// composition, which humans choose less than tools assume.
	score -= 6 * coreDigits
	score -= 7 * coreSymbols

	// Trailing digits: 1–4 (a year, "1", "123") is the most common suffix.
	switch {
	case trailingDigits >= 1 && trailingDigits <= 2:
		score += 18
	case trailingDigits >= 3 && trailingDigits <= 4:
		score += 14
	case trailingDigits == 0:
		score += 5
	default:
		score += 2 // 5+ trailing digits is unusual
	}

	// Trailing symbol: exactly one (often "!") is common; several is rare.
	switch {
	case trailingSymbols == 1:
		score += 12
	case trailingSymbols == 0:
		score += 5
	default:
		score += 1
	}

	// Case pattern of the core.
	switch casePattern(core) {
	case caseCapitalized:
		score += 14
	case caseLower:
		score += 12
	case caseUpper:
		score += 5
	case caseMixed:
		score -= 6
	}

	// Duplicated whole string ("johnjohn") is a generator artifact, not a habit.
	if n%2 == 0 && n >= 4 && string(r[:n/2]) == string(r[n/2:]) {
		score -= 20
	}

	return score
}

type casing int

const (
	caseNone casing = iota
	caseLower
	caseUpper
	caseCapitalized
	caseMixed
)

func casePattern(core []rune) casing {
	letters := 0
	upper := 0
	lower := 0
	firstUpper := false
	for i, c := range core {
		if !unicode.IsLetter(c) {
			continue
		}
		letters++
		if unicode.IsUpper(c) {
			upper++
			if i == 0 {
				firstUpper = true
			}
		} else {
			lower++
		}
	}
	switch {
	case letters == 0:
		return caseNone
	case upper == 0:
		return caseLower
	case lower == 0:
		return caseUpper
	case firstUpper && upper == 1:
		return caseCapitalized
	default:
		return caseMixed
	}
}

func classify(rs []rune) (letters, digits, symbols int) {
	for _, c := range rs {
		switch {
		case unicode.IsLetter(c):
			letters++
		case unicode.IsDigit(c):
			digits++
		default:
			symbols++
		}
	}
	return
}

func isSymbol(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsDigit(c)
}

// scored pairs a candidate with its score and original position for stable,
// deterministic ordering.
type scored struct {
	line  string
	score int
	idx   int
}

// less reports the ordering for the OUTPUT: highest score first, ties broken by
// original position (earliest first) so ranking is deterministic and stable.
func less(a, b scored) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	return a.idx < b.idx
}

// Order returns lines sorted best-first (highest Score first), stable on ties.
// It buffers all lines, so prefer TopK for very large inputs.
func Order(lines []string) []string {
	items := make([]scored, len(lines))
	for i, l := range lines {
		items[i] = scored{line: l, score: Score(l), idx: i}
	}
	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.line
	}
	return out
}
