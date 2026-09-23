// Package combine lazily traverses ordered combinations of distinct tokens.
package combine

import (
	"context"
	"fmt"
	"strings"
)

type Config struct {
	MaxCombine  int
	Separators  []string
	Limit       int // maximum unique results; 0 disables this bound
	MaxAttempts int // defaults to 1,000,000 visited complete paths
	MaxBytes    int // 0 disables the intermediate UTF-8 byte bound
}

type Result struct {
	Tokens      []string // populated only by Generate, not Walk
	Singles     int
	Count       int
	Attempts    int
	Oversized   int
	Capped      bool
	WorkLimited bool
}

// Generate collects Walk for compatibility with callers needing all combinations.
func Generate(toks []string, cfg Config) Result {
	var values []string
	res, _ := Walk(context.Background(), toks, cfg, func(s string) bool { values = append(values, s); return true })
	res.Tokens = values
	return res
}

// Walk preserves separator/length/lexicographic path ordering without storing the
// previous breadth level or reserving n*n nodes. false from visit stops immediately.
func Walk(ctx context.Context, toks []string, cfg Config, visit func(string) bool) (Result, error) {
	res := Result{}
	seen := map[string]struct{}{}
	maxK := cfg.MaxCombine
	if maxK < 1 {
		maxK = 1
	}
	if maxK > len(toks) {
		maxK = len(toks)
	}
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1000000
	}
	tick := func() bool {
		if ctx.Err() != nil {
			return false
		}
		if res.Attempts >= maxAttempts {
			res.WorkLimited = true
			return false
		}
		res.Attempts++
		return true
	}
	emit := func(s string, single bool) bool {
		if !tick() {
			return false
		}
		if cfg.MaxBytes > 0 && len(s) > cfg.MaxBytes {
			res.Oversized++
			return true
		}
		if _, ok := seen[s]; ok {
			return true
		}
		if cfg.Limit > 0 && res.Count >= cfg.Limit {
			res.Capped = true
			return false
		}
		seen[s] = struct{}{}
		res.Count++
		if single {
			res.Singles++
		}
		return visit(s)
	}
	for _, t := range toks {
		if !emit(t, true) {
			return res, ctx.Err()
		}
	}
	if maxK < 2 {
		return res, ctx.Err()
	}
	seps := cfg.Separators
	if len(seps) == 0 {
		seps = []string{""}
	}
	used := make([]bool, len(toks))
	path := make([]string, maxK)
	var walk func(int, int, string) bool
	walk = func(depth, target int, sep string) bool {
		if ctx.Err() != nil {
			return false
		}
		if depth == target {
			if cfg.MaxBytes > 0 {
				total := 0
				for i, part := range path[:target] {
					extra := len(part)
					if i > 0 {
						extra += len(sep)
					}
					if extra > cfg.MaxBytes-total {
						if !tick() {
							return false
						}
						res.Oversized++
						return true
					}
					total += extra
				}
			}
			return emit(strings.Join(path[:target], sep), false)
		}
		for i, t := range toks {
			if used[i] {
				continue
			}
			used[i] = true
			path[depth] = t
			ok := walk(depth+1, target, sep)
			used[i] = false
			if !ok {
				return false
			}
		}
		return true
	}
	for _, sep := range seps {
		for k := 2; k <= maxK; k++ {
			if !walk(0, k, sep) {
				return res, ctx.Err()
			}
		}
	}
	return res, ctx.Err()
}

func (r Result) Describe() string {
	s := fmt.Sprintf("combined=%d (singles=%d)", r.Count, r.Singles)
	if r.Capped {
		s += " [combine LIMIT reached]"
	}
	if r.WorkLimited {
		s += " [combine WORK LIMIT reached]"
	}
	if r.Oversized > 0 {
		s += fmt.Sprintf(" [%d combinations exceeded byte limit]", r.Oversized)
	}
	return s
}
