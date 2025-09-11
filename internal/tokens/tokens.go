// Package tokens turns a validated Profile into a de-duplicated set of base
// tokens: the raw building blocks that later stages combine and mutate. It
// deliberately does NOT case-explode or leet here — that is the mutation
// stage's job. Tokens are emitted lowercase and in a stable, deterministic
// order so output is reproducible.
package tokens

import (
	"sort"
	"strings"
	"time"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

// Build derives the full base-token set for a run: the profile-derived tokens
// from Extract, plus opt-in keyboard-walk seeds when rules.keyboard_walks is
// set. The result stays deterministic (unique + sorted) so downstream stages
// and tests see stable output regardless of which sources contributed.
func Build(cfg *profile.Config) []string {
	base := Extract(cfg.Profile)
	if cfg == nil || !cfg.Rules.KeyboardWalks {
		return base
	}
	set := newOrderedSet()
	for _, t := range base {
		set.add(t)
	}
	for _, t := range keyboardWalkSeeds {
		set.add(t)
	}
	out := set.slice()
	sort.Strings(out)
	return out
}

// Extract derives base tokens from a profile. The result is deterministic:
// unique, lowercased, and sorted for stable output and testability.
func Extract(p profile.Profile) []string {
	set := newOrderedSet()

	// addFolded adds a lowercase word plus any ASCII-folded variants it has, so
	// "josé" also yields "jose" and "müller" yields "muller" and "mueller".
	addFolded := func(w string) {
		w = strings.ToLower(w)
		set.add(w)
		for _, f := range foldVariants(w) {
			set.add(f)
		}
	}

	addWord := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		// A raw multi-word value yields: the split words, the glued form, and
		// separated joins ("Acme Corp" -> acme, corp, acmecorp, acme_corp).
		words := splitWords(s)
		for _, w := range words {
			addFolded(w)
		}
		if len(words) > 1 {
			addFolded(strings.Join(words, ""))
			addFolded(strings.Join(words, "_"))
		}
	}

	addWord(p.FirstName)
	addWord(p.LastName)
	addWord(p.Nickname)
	addWord(p.Partner)
	addWord(p.Company)
	for _, pet := range p.Pets {
		addWord(pet)
	}
	for _, kw := range p.Keywords {
		addWord(kw)
	}

	// Domain: keep the second-level label (acme.com -> acme) plus the full host.
	if p.Domain != "" {
		host := strings.ToLower(strings.TrimSpace(p.Domain))
		set.add(host)
		if label := domainLabel(host); label != "" {
			set.add(label)
		}
	}

	for _, d := range decomposeDate(p.DOB) {
		set.add(d)
	}

	out := set.slice()
	sort.Strings(out)
	return out
}

// splitWords breaks a value on whitespace and common separators.
func splitWords(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', '.', '_', '-', '@', '/', ',':
			return true
		}
		return false
	})
	return fields
}

// domainLabel returns the registrable-ish label of a host (best-effort, no
// network / PSL): the second-to-last dotted segment, else the first.
func domainLabel(host string) string {
	parts := strings.Split(host, ".")
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return parts[len(parts)-2]
	}
}

// decomposeDate expands a YYYY-MM-DD date into common password fragments:
//
//	1990-05-12 -> 1990, 90, 05, 12, 5, 0512, 1205, 12051990, 05121990, 051290
//
// Invalid or empty dates yield nothing.
func decomposeDate(dob string) []string {
	dob = strings.TrimSpace(dob)
	if dob == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return nil
	}
	yyyy := t.Format("2006")
	yy := t.Format("06")
	mm := t.Format("01")
	dd := t.Format("02")
	m := strings.TrimPrefix(mm, "0")
	d := strings.TrimPrefix(dd, "0")

	frags := []string{
		yyyy, yy, mm, dd, m, d,
		mm + dd,        // 0512
		dd + mm,        // 1205
		m + d,          // 512  (unpadded)
		d + m,          // 125  (unpadded)
		mm + yy,        // 0590
		yy + mm,        // 9005
		mm + yyyy,      // 051990
		yyyy + mm,      // 199005
		dd + mm + yyyy, // 12051990
		mm + dd + yyyy, // 05121990
		dd + mm + yy,   // 120590
		mm + dd + yy,   // 051290
		yyyy + mm + dd, // 19900512
		yy + mm + dd,   // 900512
	}
	// Drop empties / accidental duplicates while preserving determinism upstream.
	seen := make(map[string]struct{}, len(frags))
	out := frags[:0]
	for _, f := range frags {
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// orderedSet is a small insertion-order-agnostic unique string set. Order is
// imposed by the caller (sorted), so this only guarantees uniqueness.
type orderedSet struct {
	seen map[string]struct{}
	vals []string
}

func newOrderedSet() *orderedSet {
	return &orderedSet{seen: make(map[string]struct{})}
}

func (s *orderedSet) add(v string) {
	if v == "" {
		return
	}
	if _, ok := s.seen[v]; ok {
		return
	}
	s.seen[v] = struct{}{}
	s.vals = append(s.vals, v)
}

func (s *orderedSet) slice() []string {
	out := make([]string, len(s.vals))
	copy(out, s.vals)
	return out
}
