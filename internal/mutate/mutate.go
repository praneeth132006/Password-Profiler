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

// Config controls a mutation run.
type Config struct {
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

// Expand produces the candidate list for a single base token. The output is
// deterministic and de-duplicated: case seeds first (in configured order),
// then breadth-first chained mutations up to the depth cap.
func (e *Engine) Expand(token string) []string {
	if token == "" {
		return nil
	}

	result := make([]string, 0, 16)
	seen := make(map[string]struct{})
	add := func(s string) bool {
		if s == "" {
			return false
		}
		if _, ok := seen[s]; ok {
			return false
		}
		seen[s] = struct{}{}
		result = append(result, s)
		return true
	}

	// Case seeds are candidates in their own right and the roots of chaining.
	var frontier []string
	for _, cf := range e.cases {
		s := cf(token)
		if add(s) {
			frontier = append(frontier, s)
		}
	}

	// Breadth-first chaining: each level applies one more atomic mutation.
	for level := 0; level < e.depth && len(frontier) > 0; level++ {
		var next []string
		for _, s := range frontier {
			for _, child := range e.atomic(s) {
				if add(child) {
					next = append(next, child)
				}
			}
		}
		frontier = next
	}
	return result
}

// atomic returns every single-step mutation of s (never s itself), in a
// deterministic family order: leet, append, prepend, structural.
func (e *Engine) atomic(s string) []string {
	out := make([]string, 0, 8)
	out = append(out, leetVariants(s, e.cfg.Leet, e.cfg.LeetCap)...)
	for _, a := range e.cfg.Append {
		if a != "" {
			out = append(out, s+a)
		}
	}
	for _, p := range e.cfg.Prepend {
		if p != "" {
			out = append(out, p+s)
		}
	}
	if e.cfg.Structural {
		out = append(out, structuralVariants(s)...)
	}
	return out
}
