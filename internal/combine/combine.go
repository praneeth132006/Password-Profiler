// Package combine expands base tokens into a larger token set by joining them
// together: the singles unchanged, then bounded ordered concatenations of
// distinct tokens using a configurable separator set (e.g. "", ".", "_").
//
// It sits between the tokens and mutate stages of the pipeline. Combination is
// the fastest-exploding stage, so it is bounded two ways: MaxCombine caps how
// many tokens may be joined, and Limit caps the total number of emitted
// combinations (a naive run must never fill memory before the output budget
// even applies). Output is deterministic and de-duplicated.
package combine

import "fmt"

// Config controls combination.
type Config struct {
	// MaxCombine is the maximum number of tokens joined into one combination.
	// 1 => singles only. Values < 1 are treated as 1.
	MaxCombine int
	// Separators are inserted between joined tokens. Each separator produces
	// its own set of combinations. Empty => {""} (bare concatenation).
	Separators []string
	// Limit caps the number of emitted combinations. 0 => unlimited. When the
	// limit is reached, generation stops cleanly and Result.Capped is true.
	Limit int
}

// Result carries the combined tokens plus a report of what happened.
type Result struct {
	Tokens  []string // combined tokens, deterministic and de-duplicated
	Singles int      // number of length-1 tokens emitted
	Capped  bool     // true if Limit stopped generation early
}

// Generate expands toks according to cfg. toks is expected to already be
// unique and sorted (as produced by the tokens stage); that ordering, together
// with the separator order, makes the output deterministic.
func Generate(toks []string, cfg Config) Result {
	maxK := cfg.MaxCombine
	if maxK < 1 {
		maxK = 1
	}
	seps := cfg.Separators
	if len(seps) == 0 {
		seps = []string{""}
	}

	res := Result{}
	seen := make(map[string]struct{})
	// add returns false once the limit is reached, signalling callers to stop.
	add := func(s string) bool {
		if _, ok := seen[s]; ok {
			return true
		}
		if cfg.Limit > 0 && len(res.Tokens) >= cfg.Limit {
			res.Capped = true
			return false
		}
		seen[s] = struct{}{}
		res.Tokens = append(res.Tokens, s)
		return true
	}

	// Singles first — separator-independent, emitted exactly once.
	for _, t := range toks {
		if !add(t) {
			return res
		}
		res.Singles++
	}

	if maxK < 2 || len(toks) < 2 {
		return res
	}

	// For each separator, breadth-first extend ordered paths of distinct token
	// indices from length 1 up to maxK, carrying the joined string so each
	// extension is a single concatenation.
	type node struct {
		idxs []int
		str  string
	}
	for _, sep := range seps {
		level := make([]node, 0, len(toks))
		for i, t := range toks {
			level = append(level, node{idxs: []int{i}, str: t})
		}
		for k := 2; k <= maxK; k++ {
			next := make([]node, 0, len(level)*len(toks))
			for _, nd := range level {
				for j := range toks {
					if containsIdx(nd.idxs, j) {
						continue
					}
					joined := nd.str + sep + toks[j]
					if !add(joined) {
						return res
					}
					child := node{
						idxs: append(append(make([]int, 0, len(nd.idxs)+1), nd.idxs...), j),
						str:  joined,
					}
					next = append(next, child)
				}
			}
			level = next
			if len(level) == 0 {
				break
			}
		}
	}
	return res
}

func containsIdx(idxs []int, j int) bool {
	for _, i := range idxs {
		if i == j {
			return true
		}
	}
	return false
}

// Describe returns a short human-readable summary of a Result for stats output.
func (r Result) Describe() string {
	s := fmt.Sprintf("combined=%d (singles=%d)", len(r.Tokens), r.Singles)
	if r.Capped {
		s += " [combine LIMIT reached]"
	}
	return s
}
