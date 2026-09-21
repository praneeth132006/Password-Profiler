package tokens

import (
	"reflect"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func contains(t *testing.T, got []string, want ...string) {
	t.Helper()
	set := make(map[string]struct{}, len(got))
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			t.Errorf("expected token %q in %v", w, got)
		}
	}
}

func absent(t *testing.T, got []string, notWant ...string) {
	t.Helper()
	for _, g := range got {
		for _, nw := range notWant {
			if g == nw {
				t.Errorf("did not expect token %q", nw)
			}
		}
	}
}

func TestExtract(t *testing.T) {
	p := profile.Profile{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "JD",
		Company:   "Acme Corp",
		Domain:    "acme.com",
		Pets:      []string{"Rex"},
		Keywords:  []string{"Falcons"},
	}
	got := Extract(p)

	// Lowercased singles.
	contains(t, got, "john", "doe", "jd", "rex", "falcons")
	// Multi-word company split + glued + underscored.
	contains(t, got, "acme", "corp", "acmecorp", "acme_corp")
	// Domain host + label.
	contains(t, got, "acme.com", "acme")
	// No uppercase should leak from extraction (case-explosion is mutate's job).
	absent(t, got, "John", "ACME")

	// Determinism: sorted + stable across calls.
	if !reflect.DeepEqual(got, Extract(p)) {
		t.Errorf("Extract not deterministic")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Errorf("output not sorted at %d: %q > %q", i, got[i-1], got[i])
		}
	}
}

func TestDecomposeDate(t *testing.T) {
	tests := []struct {
		name string
		dob  string
		want []string // must all be present
		none bool     // expect empty result
	}{
		{
			name: "standard dob",
			dob:  "1990-05-12",
			want: []string{"1990", "90", "05", "12", "5", "0512", "1205", "12051990", "051290"},
		},
		{name: "empty", dob: "", none: true},
		{name: "invalid", dob: "not-a-date", none: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decomposeDate(tt.dob)
			if tt.none {
				if len(got) != 0 {
					t.Fatalf("expected no fragments, got %v", got)
				}
				return
			}
			contains(t, got, tt.want...)
			// Fragments must be unique.
			seen := map[string]struct{}{}
			for _, f := range got {
				if _, dup := seen[f]; dup {
					t.Errorf("duplicate fragment %q", f)
				}
				seen[f] = struct{}{}
			}
		})
	}
}

func TestSplitWords(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"Acme Corp", []string{"Acme", "Corp"}},
		{"jane_doe", []string{"jane", "doe"}},
		{"one.two-three", []string{"one", "two", "three"}},
		{"solo", []string{"solo"}},
	}
	for _, tt := range tests {
		if got := splitWords(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitWords(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestDomainLabel(t *testing.T) {
	tests := map[string]string{
		"acme.com":        "acme",
		"mail.acme.co.uk": "co",
		"localhost":       "localhost",
		"":                "",
	}
	for in, want := range tests {
		if got := domainLabel(in); got != want {
			t.Errorf("domainLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
