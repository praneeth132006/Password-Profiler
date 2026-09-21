package combine

import (
	"reflect"
	"sort"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name string
		toks []string
		cfg  Config
		want []string // exact, order-sensitive
	}{
		{
			name: "singles only when max_combine 1",
			toks: []string{"a", "b", "c"},
			cfg:  Config{MaxCombine: 1, Separators: []string{"", "_"}},
			want: []string{"a", "b", "c"},
		},
		{
			name: "pairs with single empty separator",
			toks: []string{"a", "b"},
			cfg:  Config{MaxCombine: 2, Separators: []string{""}},
			// singles, then ordered pairs a+b, b+a
			want: []string{"a", "b", "ab", "ba"},
		},
		{
			name: "pairs across multiple separators",
			toks: []string{"a", "b"},
			cfg:  Config{MaxCombine: 2, Separators: []string{"", "_"}},
			want: []string{"a", "b", "ab", "ba", "a_b", "b_a"},
		},
		{
			name: "triples ordered distinct",
			toks: []string{"a", "b", "c"},
			cfg:  Config{MaxCombine: 3, Separators: []string{"-"}},
			want: []string{
				"a", "b", "c",
				// length-2 (BFS from a,b,c)
				"a-b", "a-c", "b-a", "b-c", "c-a", "c-b",
				// length-3
				"a-b-c", "a-c-b", "b-a-c", "b-c-a", "c-a-b", "c-b-a",
			},
		},
		{
			name: "empty separators defaults to bare concat",
			toks: []string{"x", "y"},
			cfg:  Config{MaxCombine: 2, Separators: nil},
			want: []string{"x", "y", "xy", "yx"},
		},
		{
			name: "max_combine < 1 treated as 1",
			toks: []string{"a", "b"},
			cfg:  Config{MaxCombine: 0},
			want: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Generate(tt.toks, tt.cfg)
			if !reflect.DeepEqual(got.Tokens, tt.want) {
				t.Errorf("Tokens = %v, want %v", got.Tokens, tt.want)
			}
			if got.Singles != len(tt.toks) {
				t.Errorf("Singles = %d, want %d", got.Singles, len(tt.toks))
			}
			if got.Capped {
				t.Errorf("unexpected Capped=true")
			}
		})
	}
}

func TestGenerateNoDistinctSelfJoin(t *testing.T) {
	// A token must never be joined with itself.
	got := Generate([]string{"a", "b"}, Config{MaxCombine: 2, Separators: []string{""}})
	for _, tok := range got.Tokens {
		if tok == "aa" || tok == "bb" {
			t.Errorf("token combined with itself: %q", tok)
		}
	}
}

func TestGenerateLimitCaps(t *testing.T) {
	toks := []string{"a", "b", "c", "d"}
	got := Generate(toks, Config{MaxCombine: 3, Separators: []string{"", "_"}, Limit: 5})
	if !got.Capped {
		t.Fatalf("expected Capped=true")
	}
	if len(got.Tokens) != 5 {
		t.Errorf("emitted %d tokens, want exactly 5 (limit)", len(got.Tokens))
	}
	// Singles must survive the cap (they are emitted first).
	if got.Singles != 4 {
		t.Errorf("Singles = %d, want 4", got.Singles)
	}
}

func TestGenerateDeterministicAndUnique(t *testing.T) {
	toks := []string{"acme", "corp", "jd"}
	cfg := Config{MaxCombine: 2, Separators: []string{"", "_", "."}}
	a := Generate(toks, cfg)
	b := Generate(toks, cfg)
	if !reflect.DeepEqual(a.Tokens, b.Tokens) {
		t.Fatalf("non-deterministic output")
	}
	// Uniqueness.
	sorted := append([]string(nil), a.Tokens...)
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			t.Errorf("duplicate token %q", sorted[i])
		}
	}
}
