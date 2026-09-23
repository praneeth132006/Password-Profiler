package mutate

import (
	"context"
	"strings"
)

// LeetPair is one lowercase base rune and its leet substitutions, primary first.
type LeetPair struct {
	Base rune
	Subs []rune
}

// leetTable is the ordered, single source of truth for leet substitutions.
// Order is fixed so rule generation and mutation are deterministic and agree.
// The primary (index 0) is used by "partial" mode; "full" mode considers all.
var leetTable = []LeetPair{
	{'a', []rune{'@', '4'}},
	{'e', []rune{'3'}},
	{'i', []rune{'1', '!'}},
	{'o', []rune{'0'}},
	{'s', []rune{'$', '5'}},
	{'t', []rune{'7'}},
}

// leetSubs is the map view of leetTable for O(1) lookup during mutation.
var leetSubs = func() map[rune][]rune {
	m := make(map[rune][]rune, len(leetTable))
	for _, p := range leetTable {
		m[p.Base] = p.Subs
	}
	return m
}()

// LeetTable returns the leet substitutions in deterministic order. Other
// packages (e.g. rules) use it to encode the same substitutions as hashcat
// rules.
func LeetTable() []LeetPair { return leetTable }

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
		if cap == 0 {
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

// walkLeet propagates cancellation/consumer stop through full-leet enumeration;
// it never allocates a slice containing all combinations.
func walkLeet(ctx context.Context, s, mode string, cap int, visit func(string) bool) bool {
	switch mode {
	case "partial":
		for _, v := range leetPartial(s) {
			if ctx.Err() != nil || !visit(v) {
				return false
			}
		}
		return true
	case "full":
		if cap == 0 {
			cap = defaultLeetCap
		}
		return walkLeetFull(ctx, s, cap, visit)
	default:
		return ctx.Err() == nil
	}
}

func leetFull(s string, cap int) []string {
	var out []string
	walkLeetFull(context.Background(), s, cap, func(v string) bool { out = append(out, v); return true })
	return out
}

func walkLeetFull(ctx context.Context, s string, cap int, visit func(string) bool) bool {
	runes := []rune(s)
	var positions []int
	for i, r := range runes {
		if _, ok := leetSubs[leetKey(r)]; ok {
			positions = append(positions, i)
		}
	}
	if len(positions) == 0 {
		return true
	}
	work := append([]rune(nil), runes...)
	var recurse func(int, int, bool) bool
	recurse = func(pi, left int, changed bool) bool {
		if ctx.Err() != nil {
			return false
		}
		if pi == len(positions) || left == 0 {
			if changed {
				return visit(string(work))
			}
			return true
		}
		pos := positions[pi]
		orig := work[pos]
		if !recurse(pi+1, left, changed) {
			return false
		}
		for _, sub := range leetSubs[leetKey(orig)] {
			work[pos] = sub
			if !recurse(pi+1, left-1, true) {
				work[pos] = orig
				return false
			}
		}
		work[pos] = orig
		return true
	}
	if cap < 0 {
		cap = len(positions)
	}
	return recurse(0, cap, false)
}
