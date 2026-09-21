package mutate

import (
	"reflect"
	"sort"
	"testing"
)

func TestEngineExpandCaseAndAffix(t *testing.T) {
	// depth 1, no leet, no structural: reproduces "case + append suffix".
	eng := NewEngine(Config{
		Cases:  []string{"lower", "capitalize"},
		Append: []string{"123", "!"},
		Depth:  1,
	})
	got := eng.Expand("rex")
	want := []string{"rex", "Rex", "rex123", "rex!", "Rex123", "Rex!"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand = %v, want %v", got, want)
	}
}

func TestEngineExpandPrepend(t *testing.T) {
	eng := NewEngine(Config{
		Cases:   []string{"lower"},
		Prepend: []string{"2024"},
		Append:  []string{"!"},
		Depth:   1,
	})
	got := eng.Expand("john")
	want := []string{"john", "john!", "2024john"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand = %v, want %v", got, want)
	}
}

func TestEngineExpandStructural(t *testing.T) {
	eng := NewEngine(Config{
		Cases:      []string{"lower"},
		Structural: true,
		Depth:      1,
	})
	got := eng.Expand("abc")
	// seed abc, then reverse(cba), duplicate(abcabc), truncate(ab)
	want := []string{"abc", "cba", "abcabc", "ab"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand = %v, want %v", got, want)
	}
}

func TestEngineDepthChaining(t *testing.T) {
	// depth 2 should allow stacking two appends: rex -> rex1 -> rex12.
	eng := NewEngine(Config{
		Cases:  []string{"lower"},
		Append: []string{"1", "2"},
		Depth:  2,
	})
	got := eng.Expand("rex")
	// Depth 2 must include a doubly-affixed candidate.
	if !contains(got, "rex12") || !contains(got, "rex21") {
		t.Errorf("expected depth-2 stacked affixes in %v", got)
	}
	// Depth 1 must NOT.
	eng1 := NewEngine(Config{Cases: []string{"lower"}, Append: []string{"1", "2"}, Depth: 1})
	if contains(eng1.Expand("rex"), "rex12") {
		t.Errorf("depth 1 must not stack affixes")
	}
}

func TestEngineDedupAndDeterminism(t *testing.T) {
	cfg := Config{Cases: []string{"lower", "identity"}, Append: []string{"1"}, Leet: "partial", Structural: true, Depth: 2}
	eng := NewEngine(cfg)
	a := eng.Expand("test")
	b := eng.Expand("test")
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("non-deterministic output")
	}
	sorted := append([]string(nil), a...)
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			t.Errorf("duplicate candidate %q", sorted[i])
		}
	}
}

func TestEngineEmptyToken(t *testing.T) {
	eng := NewEngine(Config{Cases: []string{"lower"}})
	if got := eng.Expand(""); got != nil {
		t.Errorf("Expand(\"\") = %v, want nil", got)
	}
}

func TestEngineNoCasesFallsBackToIdentity(t *testing.T) {
	eng := NewEngine(Config{Cases: nil, Append: []string{"9"}, Depth: 1})
	got := eng.Expand("kite")
	want := []string{"kite", "kite9"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expand = %v, want %v", got, want)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
