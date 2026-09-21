package mutate

import (
	"sort"
	"testing"
)

func sortedCopy(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func TestLeetVariantsOff(t *testing.T) {
	if got := leetVariants("password", "off", 0); got != nil {
		t.Errorf("off mode = %v, want nil", got)
	}
	if got := leetVariants("password", "bogus", 0); got != nil {
		t.Errorf("unknown mode = %v, want nil", got)
	}
}

func TestLeetPartial(t *testing.T) {
	// leetable in "test": t->7, e->3, s->$
	got := sortedCopy(leetVariants("test", "partial", 0))
	want := sortedCopy([]string{"7es7", "t3st", "te$t", "73$7"})
	if !equal(got, want) {
		t.Errorf("partial(test) = %v, want %v", got, want)
	}
	// Word with no leetable chars yields nothing.
	if got := leetVariants("bxy", "partial", 0); got != nil {
		t.Errorf("no-leet word = %v, want nil", got)
	}
	// Never includes the original.
	for _, v := range leetVariants("test", "partial", 0) {
		if v == "test" {
			t.Errorf("partial must not include original")
		}
	}
}

func TestLeetPartialCaseInsensitive(t *testing.T) {
	// Uppercase letters still leet ("PASS" -> P@SS / PA$$ / P@$$).
	got := sortedCopy(leetVariants("PASS", "partial", 0))
	want := sortedCopy([]string{"P@SS", "PA$$", "P@$$"})
	if !equal(got, want) {
		t.Errorf("partial(PASS) = %v, want %v", got, want)
	}
}

func TestLeetFullBoundedByCap(t *testing.T) {
	// "test" has 4 leetable positions (t,e,s,t). With cap 1, at most one
	// position is substituted per variant.
	got := leetVariants("test", "full", 1)
	for _, v := range got {
		if diffPositions("test", v) > 1 {
			t.Errorf("cap 1 violated by %q", v)
		}
	}
	// Includes single-substitution variants.
	if !contains(got, "7est") || !contains(got, "te$t") {
		t.Errorf("expected single-sub variants in %v", got)
	}
	// Higher cap yields strictly more (combinatorial) variants.
	if len(leetVariants("test", "full", 3)) <= len(got) {
		t.Errorf("cap 3 should produce more variants than cap 1")
	}
	// Uses alternative substitutions too (s has $ and 5).
	if !contains(leetVariants("test", "full", 1), "te5t") {
		t.Errorf("expected alternative sub te5t")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffPositions(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) != len(rb) {
		return 1 << 30
	}
	n := 0
	for i := range ra {
		if ra[i] != rb[i] {
			n++
		}
	}
	return n
}
