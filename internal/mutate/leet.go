package mutate

import "strings"

// leetSubs maps a lowercase base rune to its leet substitutions, primary first.
// The primary (index 0) is used by "partial" mode; "full" mode considers all.
var leetSubs = map[rune][]rune{
	'a': {'@', '4'},
	'e': {'3'},
	'i': {'1', '!'},
	'o': {'0'},
	's': {'$', '5'},
	't': {'7'},
}

// defaultLeetCap bounds how many character positions "full" leet may substitute
// in a single word. Leet is the fastest-exploding transform, so this cap is the
// primary guard against combinatorial blow-up.
const defaultLeetCap = 3

// leetKey returns the lowercase form used to look up substitutions, so leet
// applies regardless of the current case ("PASS" still leets its 's').
func leetKey(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// leetVariants returns leet mutations of s (never including s itself), in a
// deterministic order, according to mode:
//
//   - "off":     none.
//   - "partial": for each distinct leetable rune present, one variant with all
//     of its occurrences replaced by the PRIMARY substitution, plus one variant
//     with every leetable rune replaced by its primary substitution.
//   - "full":    every combination substituting between 1 and cap positions,
//     each chosen position taking any of its substitutions.
func leetVariants(s, mode string, cap int) []string {
	switch mode {
	case "partial":
		return leetPartial(s)
	case "full":
		if cap <= 0 {
			cap = defaultLeetCap
		}
		return leetFull(s, cap)
	default: // "off" or unknown
		return nil
	}
}

func leetPartial(s string) []string {
	runes := []rune(s)
	// Which distinct base runes are leetable and present, in first-seen order.
	var order []rune
	seenBase := make(map[rune]struct{})
	for _, r := range runes {
		k := leetKey(r)
		if _, ok := leetSubs[k]; !ok {
			continue
		}
		if _, dup := seenBase[k]; dup {
			continue
		}
		seenBase[k] = struct{}{}
		order = append(order, k)
	}
	if len(order) == 0 {
		return nil
	}

	out := make([]string, 0, len(order)+1)
	seen := make(map[string]struct{})
	emit := func(v string) {
		if v == s {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	// One variant per distinct leetable rune (all its occurrences -> primary).
	for _, base := range order {
		primary := leetSubs[base][0]
		emit(substituteRune(runes, base, primary))
	}
	// One variant with every leetable rune replaced by its primary.
	if len(order) > 1 {
		emit(substituteAllPrimary(runes))
	}
	return out
}

// substituteRune replaces every rune whose base form equals base with sub.
func substituteRune(runes []rune, base, sub rune) string {
	var b strings.Builder
	b.Grow(len(runes))
	for _, r := range runes {
		if leetKey(r) == base {
			b.WriteRune(sub)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func substituteAllPrimary(runes []rune) string {
	var b strings.Builder
	b.Grow(len(runes))
	for _, r := range runes {
		if subs, ok := leetSubs[leetKey(r)]; ok {
			b.WriteRune(subs[0])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func leetFull(s string, cap int) []string {
	runes := []rune(s)
	// Positions that can be substituted.
	var positions []int
	for i, r := range runes {
		if _, ok := leetSubs[leetKey(r)]; ok {
			positions = append(positions, i)
		}
	}
	if len(positions) == 0 {
		return nil
	}

	out := make([]string, 0)
	seen := make(map[string]struct{})
	work := make([]rune, len(runes))
	copy(work, runes)

	var recurse func(pi, subsLeft int)
	recurse = func(pi, subsLeft int) {
		if pi == len(positions) {
			v := string(work)
			if v == s {
				return
			}
			if _, ok := seen[v]; ok {
				return
			}
			seen[v] = struct{}{}
			out = append(out, v)
			return
		}
		pos := positions[pi]
		orig := work[pos]
		// Keep original at this position.
		recurse(pi+1, subsLeft)
		// Or substitute, if budget remains.
		if subsLeft > 0 {
			for _, sub := range leetSubs[leetKey(orig)] {
				work[pos] = sub
				recurse(pi+1, subsLeft-1)
			}
			work[pos] = orig
		}
	}
	recurse(0, cap)
	return out
}
