package tokens

import (
	"reflect"
	"sort"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func TestFoldSimple(t *testing.T) {
	cases := map[string]string{
		"josé":     "jose",
		"müller":   "muller",
		"renée":    "renee",
		"françois": "francois",
		"nuñez":    "nunez",
		"ørsted":   "orsted",
		"straße":   "strasse", // ß has no single base -> expands in simple fold too
		"ascii":    "ascii",   // untouched
	}
	for in, want := range cases {
		if got := foldSimple(in); got != want {
			t.Errorf("foldSimple(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldDigraph(t *testing.T) {
	cases := map[string]string{
		"müller":      "mueller",
		"schrödinger": "schroedinger",
		"käse":        "kaese",
		"straße":      "strasse",
		"ascii":       "ascii",
	}
	for in, want := range cases {
		if got := foldDigraph(in); got != want {
			t.Errorf("foldDigraph(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFoldVariants(t *testing.T) {
	// Pure ASCII contributes nothing.
	if got := foldVariants("acme"); got != nil {
		t.Errorf("foldVariants(ascii) = %v, want nil", got)
	}
	// A plain accent yields one variant.
	if got := foldVariants("josé"); !reflect.DeepEqual(got, []string{"jose"}) {
		t.Errorf("foldVariants(josé) = %v, want [jose]", got)
	}
	// An umlaut yields both the simple and digraph forms.
	got := foldVariants("müller")
	sort.Strings(got)
	want := []string{"muller", "mueller"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("foldVariants(müller) = %v, want %v", got, want)
	}
}

func TestExtractFolding(t *testing.T) {
	p := profile.Profile{FirstName: "José", LastName: "Müller"}
	got := Extract(p)
	// Original lowercased forms plus folded ASCII variants are all present.
	contains(t, got, "josé", "jose", "müller", "muller", "mueller")
}
