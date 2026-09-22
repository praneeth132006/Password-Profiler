// Package policy filters candidate passwords against a target password policy:
// minimum/maximum length and required character classes. It lets pwprofiler
// emit only candidates that could actually be valid under the target's rules,
// shrinking the keyspace before it ever reaches the cracker.
package policy

import (
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"unicode"
	"unicode/utf8"
)

// Policy is the target password policy (mirrors the config's policy block).
type Policy struct {
	MaxBytes, MaxRepeat int
	Forbidden           string
	Blocklist           []string
	MinLen              int      // 0 => no minimum
	MaxLen              int      // 0 => no maximum
	Require             []string // any of: upper, lower, digit, special
}

// Filter is a compiled, allocation-free policy checker.
type Filter struct {
	maxBytes, maxRepeat                          int
	forbidden                                    map[rune]bool
	blocked                                      map[string]bool
	minLen, maxLen                               int
	needUpper, needLower, needDigit, needSpecial bool
	active                                       bool
}

// New compiles a Policy into a Filter.
func New(p Policy) *Filter {
	f := &Filter{minLen: p.MinLen, maxLen: p.MaxLen, maxBytes: p.MaxBytes, maxRepeat: p.MaxRepeat, forbidden: map[rune]bool{}, blocked: map[string]bool{}}
	for _, r := range p.Forbidden {
		f.forbidden[r] = true
	}
	for _, s := range p.Blocklist {
		f.blocked[s] = true
	}
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
	f.active = p.MaxBytes > 0 || p.MaxRepeat > 0 || p.Forbidden != "" || len(p.Blocklist) > 0 || p.MinLen > 0 || p.MaxLen > 0 ||
		f.needUpper || f.needLower || f.needDigit || f.needSpecial
	return f
}

// Active reports whether any constraint is set. When false, Allow is always
// true and callers can skip the filter entirely.
func (f *Filter) Active() bool { return f.active }

// Allow reports whether s satisfies the policy. Length is measured in runes.
func (f *Filter) Allow(s string) bool { return f.Reason(s) == "" }

// Reason returns the first failed constraint, never the password itself.
// A blank reason means the password satisfies this policy, not that it is strong.
func (f *Filter) Reason(s string) string {
	if !utf8.ValidString(s) {
		return "invalid_utf8"
	}
	if f.blocked[s] {
		return "blocklisted"
	}
	if f.maxBytes > 0 && len(s) > f.maxBytes {
		return "max_bytes"
	}
	n, repeat := 0, 0
	var previous rune
	var upper, lower, digit, special bool
	for _, r := range s {
		if f.forbidden[r] {
			return "forbidden_character"
		}
		n++
		if n > 1 && r == previous {
			repeat++
		} else {
			repeat = 1
		}
		previous = r
		if f.maxRepeat > 0 && repeat > f.maxRepeat {
			return "max_repeat"
		}
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case isSpecial(r):
			special = true
		}
	}
	if n < f.minLen {
		return "min_length"
	}
	if f.maxLen > 0 && n > f.maxLen {
		return "max_length"
	}
	if f.needUpper && !upper {
		return "missing_upper"
	}
	if f.needLower && !lower {
		return "missing_lower"
	}
	if f.needDigit && !digit {
		return "missing_digit"
	}
	if f.needSpecial && !special {
		return "missing_special"
	}
	return ""
}

// FromProfile keeps all entry points on the same policy implementation.
func FromProfile(p profile.Policy) *Filter {
	return New(Policy{MinLen: p.MinLen, MaxLen: p.MaxLen, Require: p.Require, MaxBytes: p.MaxBytes, MaxRepeat: p.MaxRepeat, Forbidden: p.Forbidden, Blocklist: p.Blocklist})
}

// isSpecial treats any non-space, non-alphanumeric rune as a special character
// (punctuation and symbols, e.g. ! @ # $ _ . -).
func isSpecial(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
		return false
	}
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}
