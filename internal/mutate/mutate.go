// Package mutate is the core value of pwprofiler: it turns base tokens into
// candidate passwords via composable transforms.
//
// Phase 1 implements the two simplest, highest-signal transform families —
// case folding and suffix appending. The building blocks (case functions,
// the Engine, and its config) are shaped so later phases can layer on
// leetspeak, structural mutations, and depth-bounded chaining without
// reworking callers.
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

// Config controls the Phase 1 expansion.
type Config struct {
	// Cases are the case transforms to apply, by name. Unknown names are
	// ignored (validation happens in the profile package).
	Cases []string
	// Suffixes are appended to each cased form. The empty (no-suffix) variant
	// is always emitted as well.
	Suffixes []string
}

// Engine expands tokens into candidates according to Config.
type Engine struct {
	cfg   Config
	cases []CaseFunc
}

// NewEngine resolves the configured case names into functions once.
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
	return &Engine{cfg: cfg, cases: cases}
}

// Expand produces the candidate list for a single base token. Ordering is
// deterministic: case variants in configured order, each followed by its
// no-suffix form then each configured suffix.
func (e *Engine) Expand(token string) []string {
	if token == "" {
		return nil
	}
	// Upper bound: cases * (1 + suffixes).
	out := make([]string, 0, len(e.cases)*(1+len(e.cfg.Suffixes)))
	seen := make(map[string]struct{})
	emit := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, cf := range e.cases {
		base := cf(token)
		emit(base)
		for _, suf := range e.cfg.Suffixes {
			if suf == "" {
				continue
			}
			emit(base + suf)
		}
	}
	return out
}
