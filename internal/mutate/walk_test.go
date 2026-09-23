package mutate

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// Independent breadth-level reference for equivalence testing on small trees.
func referenceExpand(e *Engine, token string) []string {
	var result, frontier []string
	seen := map[string]bool{}
	add := func(s string) bool {
		if s == "" || seen[s] {
			return false
		}
		seen[s] = true
		result = append(result, s)
		return true
	}
	for _, cf := range e.cases {
		v := cf(token)
		if add(v) {
			frontier = append(frontier, v)
		}
	}
	for depth := 0; depth < e.depth; depth++ {
		var next []string
		for _, s := range frontier {
			values := leetVariants(s, e.cfg.Leet, e.cfg.LeetCap)
			for _, a := range e.cfg.Append {
				if a != "" {
					values = append(values, s+a)
				}
			}
			for _, p := range e.cfg.Prepend {
				if p != "" {
					values = append(values, p+s)
				}
			}
			if e.cfg.Structural {
				values = append(values, structuralVariants(s)...)
			}
			for _, v := range values {
				if add(v) {
					next = append(next, v)
				}
			}
		}
		frontier = next
	}
	return result
}
func TestWalkStopsBeforeExpandingFullLeet(t *testing.T) {
	e := NewEngine(Config{Leet: "full", Depth: 8, Structural: true, Append: []string{"!"}})
	n := 0
	stats, err := e.Walk(context.Background(), strings.Repeat("a", 100), func(string) bool { n++; return n < 2 })
	if err != nil || n != 2 || stats.Attempts != 2 || stats.Limited {
		t.Fatalf("n=%d stats=%+v err=%v", n, stats, err)
	}
}
func TestWalkCapsAndCancellation(t *testing.T) {
	cases := []struct {
		name          string
		cfg           Config
		wantLimited   bool
		wantOversized bool
	}{
		{"states", Config{Append: []string{"1", "2"}, Depth: 8, MaxVariants: 5}, true, false},
		{"attempts", Config{Append: []string{"1", "1", "1", "1"}, Depth: 8, MaxAttempts: 3}, true, false},
		{"bytes", Config{Structural: true, Depth: 3, MaxBytes: 4}, false, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(tt.cfg)
			stats, err := e.Walk(context.Background(), "test", func(string) bool { return true })
			if err != nil || stats.Limited != tt.wantLimited || (stats.Oversized > 0) != tt.wantOversized {
				t.Fatalf("%+v %v", stats, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	e := NewEngine(Config{Leet: "full", Depth: 8})
	_, err := e.Walk(ctx, strings.Repeat("a", 100), func(string) bool { n++; cancel(); return true })
	if err != context.Canceled || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
func FuzzWalkMatchesReference(f *testing.F) {
	for _, s := range []string{"test", "Northstar", "é界", "123", "aA"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if s == "" || len(s) > 24 {
			return
		}
		e := NewEngine(Config{Cases: []string{"lower", "capitalize"}, Leet: "partial", Append: []string{"1!", "!"}, Prepend: []string{"20"}, Depth: 2, Structural: true, MaxVariants: 100000, MaxAttempts: 1000000})
		var got []string
		stats, err := e.Walk(context.Background(), s, func(v string) bool { got = append(got, v); return true })
		if err != nil || stats.Limited || stats.Oversized > 0 {
			t.Fatalf("%+v %v", stats, err)
		}
		if want := referenceExpand(e, s); !reflect.DeepEqual(got, want) {
			t.Fatalf("output ordering or coverage differs for %q", s)
		}
	})
}

func TestFullLeetStreamingMatchesReference(t *testing.T) {
	e := NewEngine(Config{Cases: []string{"lower", "capitalize"}, Leet: "full", Append: []string{"!"}, Depth: 2, MaxVariants: 100000, MaxAttempts: 1000000})
	var got []string
	stats, err := e.Walk(context.Background(), "test", func(s string) bool { got = append(got, s); return true })
	if err != nil || stats.Limited {
		t.Fatalf("%+v %v", stats, err)
	}
	if !reflect.DeepEqual(got, referenceExpand(e, "test")) {
		t.Fatal("full leet ordering or coverage changed")
	}
}

func TestExhaustiveIgnoresResourceCaps(t *testing.T) {
	e := NewEngine(Config{Exhaustive: true, MaxVariants: 1, MaxAttempts: 1, MaxBytes: 1, Append: []string{"0", "1"}, Depth: 3})
	got := map[string]bool{}
	stats, err := e.Walk(context.Background(), "xx", func(s string) bool { got[s] = true; return true })
	// Independent binary-tree enumeration: 1 + 2 + 4 + 8 states.
	if err != nil || stats.Limited || stats.Oversized != 0 || len(got) != 15 {
		t.Fatalf("%+v n=%d err=%v", stats, len(got), err)
	}
	for _, s := range []string{"xx", "xx000", "xx001", "xx010", "xx011", "xx100", "xx101", "xx110", "xx111"} {
		if !got[s] {
			t.Fatal("missing", s)
		}
	}
}
func TestAllLeetPositions(t *testing.T) {
	e := NewEngine(Config{Exhaustive: true, Leet: "full", LeetCap: -1, Depth: 1})
	got := map[string]bool{}
	_, err := e.Walk(context.Background(), "aaaa", func(s string) bool { got[s] = true; return true })
	if err != nil || len(got) != 81 {
		t.Fatalf("expected 3^4 states; got %d, %v", len(got), err)
	}
	for _, a := range "a@4" {
		for _, b := range "a@4" {
			for _, c := range "a@4" {
				for _, d := range "a@4" {
					if !got[string([]rune{a, b, c, d})] {
						t.Fatal("missing leet combination")
					}
				}
			}
		}
	}
}
