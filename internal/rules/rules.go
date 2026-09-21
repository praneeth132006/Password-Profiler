// Package rules emits hashcat-compatible .rule files that encode pwprofiler's
// mutation set. This is the headline feature: instead of materializing a giant
// wordlist, rule mode writes a small base wordlist plus a .rule file, and
// hashcat expands `base × rules` on the fly (`hashcat -r file.rule wordlist`).
//
// Each generated line is a standalone hashcat rule applied independently to
// every base word. The subset of the rule language used here — no-op, case,
// substitution, append/prepend, reverse/duplicate/truncate — is validated by
// the sibling Apply interpreter so we can prove a rule produces its intended
// candidate without shelling out to hashcat.
package rules

import (
	"bufio"
	"fmt"
	"io"

	"github.com/praneeth132006/Password-Profiler/internal/mutate"
)

// Config selects which rules to emit. It mirrors the mutation config.
type Config struct {
	Cases      []string // identity|lower|upper|capitalize|toggle|invert
	Leet       string   // off | partial | full
	Append     []string // affixes appended
	Prepend    []string // affixes prepended
	Structural bool     // reverse/duplicate/truncate
}

// caseRule maps a case name to its hashcat function.
var caseRule = map[string]string{
	"identity":   ":",
	"lower":      "l",
	"upper":      "u",
	"capitalize": "c",
	"toggle":     "T0",
	"invert":     "t",
}

// Set is an ordered, de-duplicated collection of rule lines.
type Set struct {
	lines []string
	seen  map[string]struct{}
}

func newSet() *Set { return &Set{seen: make(map[string]struct{})} }

func (s *Set) add(line string) {
	if line == "" {
		return
	}
	if _, ok := s.seen[line]; ok {
		return
	}
	s.seen[line] = struct{}{}
	s.lines = append(s.lines, line)
}

// Lines returns the rule lines in deterministic order.
func (s *Set) Lines() []string { return s.lines }

// Len reports how many rules were generated.
func (s *Set) Len() int { return len(s.lines) }

// WriteTo writes the rule file (one rule per line) through a buffered writer.
func (s *Set) WriteTo(w io.Writer) (int64, error) {
	bw := bufio.NewWriter(w)
	var n int64
	for _, line := range s.lines {
		m, err := bw.WriteString(line + "\n")
		n += int64(m)
		if err != nil {
			return n, fmt.Errorf("write rule: %w", err)
		}
	}
	if err := bw.Flush(); err != nil {
		return n, fmt.Errorf("flush rules: %w", err)
	}
	return n, nil
}

// Generate builds the rule set from cfg. Order: no-op, cases, leet, appends,
// prepends, structural — deterministic and de-duplicated.
func Generate(cfg Config) *Set {
	s := newSet()

	// Always include the no-op so base words pass through unchanged.
	s.add(":")

	for _, name := range cfg.Cases {
		if r, ok := caseRule[name]; ok {
			s.add(r)
		}
	}

	switch cfg.Leet {
	case "partial":
		var all string
		for _, p := range mutate.LeetTable() {
			primary := p.Subs[0]
			s.add(subRule(p.Base, primary))
			all += subRule(p.Base, primary)
		}
		s.add(all) // one line substituting every leetable char (primary)
	case "full":
		for _, p := range mutate.LeetTable() {
			for _, sub := range p.Subs {
				s.add(subRule(p.Base, sub))
			}
		}
	}

	for _, a := range cfg.Append {
		if line := appendRule(a); line != "" {
			s.add(line)
		}
	}
	for _, p := range cfg.Prepend {
		if line := prependRule(p); line != "" {
			s.add(line)
		}
	}

	if cfg.Structural {
		s.add("r") // reverse
		s.add("d") // duplicate
		s.add("]") // truncate last char
	}
	return s
}

// subRule encodes a single leet substitution: s<base><sub> (e.g. "sa@").
func subRule(base, sub rune) string {
	return "s" + string(base) + string(sub)
}

// appendRule encodes appending each rune of s: "123" -> "$1$2$3".
func appendRule(s string) string {
	var out string
	for _, r := range s {
		out += "$" + string(r)
	}
	return out
}

// prependRule encodes prepending s. hashcat's ^ prepends a single char to the
// front, so to prepend "1990" the chars are emitted in REVERSE: "^0^9^9^1".
func prependRule(s string) string {
	r := []rune(s)
	var out string
	for i := len(r) - 1; i >= 0; i-- {
		out += "^" + string(r[i])
	}
	return out
}
