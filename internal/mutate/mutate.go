// Package mutate is the core value of pwprofiler: it turns base tokens into
// candidate passwords via composable transforms.
//
// Transforms share the signature func(string) []string and are chained by a
// breadth-first walk bounded by a depth cap: from each token's case "seeds",
// the engine applies atomic mutations (leetspeak, affixes, structural) up to
// `depth` times in any order, de-duplicating as it goes. Explosion is
// controlled at every step — the leet substitution cap, the bounded affix
// lists, the depth cap, and (upstream) the global budget.
package mutate

import "context"

// Config controls a mutation run.
type Config struct {
	Exhaustive bool // disable work/state/byte caps; depth still defines the finite search
	// MaxVariants and MaxAttempts bound retained states and attempted mutations per token.
	// Zero values select defaults: 10,000 states, 100,000 attempts, 4,096 UTF-8 bytes.
	MaxVariants int
	MaxAttempts int
	MaxBytes    int
	// Cases are the seed case transforms applied to each token, by name. The
	// resulting cased forms are the starting points for chaining.
	Cases []string
	// Leet selects the leetspeak mode: off | partial | full.
	Leet string
	// LeetCap bounds substitutions per word in "full" mode (0 => default).
	LeetCap int
	// Append and Prepend are affix lists concatenated onto candidates during
	// chaining (e.g. years and numeric/symbol suffixes).
	Append  []string
	Prepend []string
	// Structural enables reverse/duplicate/truncate mutations in the chain.
	Structural bool
	// Depth is the maximum number of chained atomic mutations (>= 1).
	Depth int
}

// Engine expands tokens into candidates according to Config.
type Engine struct {
	cfg   Config
	cases []CaseFunc
	depth int
}

// NewEngine resolves configured case names into functions once and normalizes
// bounds.
func NewEngine(cfg Config) *Engine {
	cases := make([]CaseFunc, 0, len(cfg.Cases))
	for _, name := range cfg.Cases {
		if fn, ok := caseFuncs[name]; ok {
			cases = append(cases, fn)
		}
	}
	if len(cases) == 0 {
		cases = append(cases, Identity)
	}
	depth := cfg.Depth
	if depth < 1 {
		depth = 1
	}
	return &Engine{cfg: cfg, cases: cases, depth: depth}
}

// WalkStats records actual search work, including explicit completeness limits.
type WalkStats struct {
	Attempts  int
	Unique    int
	Limited   bool
	Oversized int
}

// Expand is the bounded compatibility collector. Prefer Walk for early termination
// and visibility into limits. Both preserve the historical breadth-first ordering.
func (e *Engine) Expand(token string) []string {
	var out []string
	_, _ = e.Walk(context.Background(), token, func(s string) bool { out = append(out, s); return true })
	return out
}

// Walk emits a unique candidate as soon as it is discovered. Returning false from
// visit stops the search immediately. Policy rejection must NOT stop exploration:
// a later transform can repair a rejected intermediate value.
func (e *Engine) Walk(ctx context.Context, token string, visit func(string) bool) (WalkStats, error) {
	stats := WalkStats{}
	maxVariants, maxAttempts, maxBytes := e.cfg.MaxVariants, e.cfg.MaxAttempts, e.cfg.MaxBytes
	if maxVariants <= 0 {
		maxVariants = 10000
	}
	if maxAttempts <= 0 {
		maxAttempts = 100000
	}
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	if token == "" {
		return stats, nil
	}
	if !e.cfg.Exhaustive && len(token) > maxBytes {
		stats.Oversized++
		return stats, nil
	}
	type node struct {
		value string
		depth int
	}
	var queue []node
	seen := map[string]struct{}{}
	add := func(s string, depth int) bool {
		if ctx.Err() != nil {
			return false
		}
		if !e.cfg.Exhaustive && stats.Attempts >= maxAttempts {
			stats.Limited = true
			return false
		}
		stats.Attempts++
		if s == "" {
			return true
		}
		if !e.cfg.Exhaustive && len(s) > maxBytes {
			stats.Oversized++
			return true
		}
		if _, ok := seen[s]; ok {
			return true
		}
		if !e.cfg.Exhaustive && stats.Unique >= maxVariants {
			stats.Limited = true
			return false
		}
		seen[s] = struct{}{}
		stats.Unique++
		if !visit(s) {
			return false
		}
		if depth < e.depth {
			queue = append(queue, node{s, depth})
		}
		return true
	}
	for _, cf := range e.cases {
		if !add(cf(token), 0) {
			return stats, ctx.Err()
		}
	}
	for head := 0; head < len(queue); head++ {
		nd := queue[head]
		queue[head] = node{}
		if !e.walkAtomic(ctx, nd.value, func(s string) bool { return add(s, nd.depth+1) }) {
			return stats, ctx.Err()
		}
	}
	return stats, ctx.Err()
}

func (e *Engine) walkAtomic(ctx context.Context, s string, visit func(string) bool) bool {
	if !walkLeet(ctx, s, e.cfg.Leet, e.cfg.LeetCap, visit) {
		return false
	}
	for _, a := range e.cfg.Append {
		if a != "" && !visit(s+a) {
			return false
		}
	}
	for _, p := range e.cfg.Prepend {
		if p != "" && !visit(p+s) {
			return false
		}
	}
	if e.cfg.Structural {
		for _, v := range structuralVariants(s) {
			if !visit(v) {
				return false
			}
		}
	}
	return ctx.Err() == nil
}
