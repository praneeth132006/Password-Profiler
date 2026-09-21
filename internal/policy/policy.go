// Package policy filters candidate passwords against a target password policy:
// minimum/maximum length and required character classes. It lets pwprofiler
// emit only candidates that could actually be valid under the target's rules,
// shrinking the keyspace before it ever reaches the cracker.
package policy

import "unicode"

// Policy is the target password policy (mirrors the config's policy block).
type Policy struct {
	MinLen  int      // 0 => no minimum
	MaxLen  int      // 0 => no maximum
	Require []string // any of: upper, lower, digit, special
}

// Filter is a compiled, allocation-free policy checker.
type Filter struct {
	minLen, maxLen                               int
	needUpper, needLower, needDigit, needSpecial bool
	active                                       bool
}

// New compiles a Policy into a Filter.
func New(p Policy) *Filter {
	f := &Filter{minLen: p.MinLen, maxLen: p.MaxLen}
	for _, r := range p.Require {
		switch r {
		case "upper":
			f.needUpper = true
		case "lower":
			f.needLower = true
		case "digit":
			f.needDigit = true
		case "special":
			f.needSpecial = true
		}
	}
	f.active = p.MinLen > 0 || p.MaxLen > 0 ||
		f.needUpper || f.needLower || f.needDigit || f.needSpecial
	return f
}

// Active reports whether any constraint is set. When false, Allow is always
// true and callers can skip the filter entirely.
func (f *Filter) Active() bool { return f.active }

// Allow reports whether s satisfies the policy. Length is measured in runes.
func (f *Filter) Allow(s string) bool {
	if !f.active {
		return true
	}

	// Single pass: count length and detect character classes.
	n := 0
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range s {
		n++
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case isSpecial(r):
			hasSpecial = true
		}
		// Early exit if we've already blown the max length.
		if f.maxLen > 0 && n > f.maxLen {
			return false
		}
	}

	if f.minLen > 0 && n < f.minLen {
		return false
	}
	if f.maxLen > 0 && n > f.maxLen {
		return false
	}
	if f.needUpper && !hasUpper {
		return false
	}
	if f.needLower && !hasLower {
		return false
	}
	if f.needDigit && !hasDigit {
		return false
	}
	if f.needSpecial && !hasSpecial {
		return false
	}
	return true
}

// isSpecial treats any non-space, non-alphanumeric rune as a special character
// (punctuation and symbols, e.g. ! @ # $ _ . -).
func isSpecial(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
		return false
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}
