package rank

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// commonSuffixes lists the trailing digit/symbol tails people append most often,
// in rough descending real-world frequency. It is a small, published-composition
// signal (structure, not a password dictionary) used to refine ordering among
// candidates that share the same shape: "Summer1" outranks "Summer4817".
var commonSuffixes = []string{
	"1", "123", "!", "12", "2", "01", "13", "7", "11", "21",
	"22", "23", "69", "007", "99", "00", "1!", "123!", "!!", "@",
}

// suffixWeights maps a tail to a small bonus (highest for the most common).
var suffixWeights = func() map[string]int {
	m := make(map[string]int, len(commonSuffixes))
	n := len(commonSuffixes)
	for i, s := range commonSuffixes {
		m[s] = (n - i) // 20..1
	}
	return m
}()

// suffixBonus returns a modest bonus (0..~10) when a candidate ends in a common
// tail. It is intentionally small so it only breaks ties between same-shape
// candidates and never overrides length/shape scoring.
func suffixBonus(pw string) int {
	tail := trailingTail(pw)
	if tail == "" {
		return 0
	}
	if w, ok := suffixWeights[tail]; ok {
		// Scale 1..20 down into 0..10.
		return (w + 1) / 2
	}
	return 0
}

// trailingTail returns the trailing run of digits and symbols (the "suffix"),
// e.g. "Falcons2024!" -> "2024!". Returns "" when the string ends in a letter.
func trailingTail(pw string) string {
	r := []rune(pw)
	i := len(r)
	for i > 0 && !unicode.IsLetter(r[i-1]) {
		i--
	}
	if i == len(r) {
		return ""
	}
	return string(r[i:])
}

// Model is an optional, user-supplied frequency corpus that boosts candidates
// whose core word or suffix appears in a target-specific list (for example, a
// frequency-ordered wordlist an auditor is permitted to use). It never lowers a
// score; it only adds evidence on top of the built-in heuristic.
type Model struct {
	weight map[string]int
	max    int
}

// LoadModel reads a frequency-ordered list, one term per line, most frequent
// first. Blank lines and lines beginning with '#' are ignored. Earlier terms
// get higher weight. Terms are compared case-insensitively.
func LoadModel(r io.Reader) (*Model, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var terms []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		terms = append(terms, strings.ToLower(line))
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read model: %w", err)
	}
	m := &Model{weight: make(map[string]int, len(terms)), max: len(terms)}
	for i, t := range terms {
		if _, seen := m.weight[t]; seen {
			continue // keep the highest (earliest) weight for a repeated term
		}
		m.weight[t] = len(terms) - i // highest for the first line
	}
	return m, nil
}

// bonus returns a score boost (0..~40) when the candidate's lowercased core or
// full form appears in the model, scaled by the term's frequency rank.
func (m *Model) bonus(pw string) int {
	if m == nil || m.max == 0 {
		return 0
	}
	lower := strings.ToLower(pw)
	best := 0
	if w, ok := m.weight[lower]; ok && w > best {
		best = w
	}
	if core := strings.ToLower(coreWord(pw)); core != "" && core != lower {
		if w, ok := m.weight[core]; ok && w > best {
			best = w
		}
	}
	if best == 0 {
		return 0
	}
	// Scale the frequency rank into 0..40 so a top model hit is decisive but a
	// mid-list hit is a nudge.
	return 1 + (best*39)/m.max
}

// Score returns the heuristic Score plus this model's frequency bonus.
func (m *Model) Score(pw string) int { return Score(pw) + m.bonus(pw) }

// coreWord returns the leading run of letters (the word part before any digits
// or symbols), e.g. "Falcons2024!" -> "Falcons".
func coreWord(pw string) string {
	r := []rune(pw)
	i := 0
	for i < len(r) && unicode.IsLetter(r[i]) {
		i++
	}
	return string(r[:i])
}
