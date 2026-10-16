package tokens

import (
	"strings"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func TestCommonSeeds(t *testing.T) {
	if CommonCount() < 50 {
		t.Fatalf("bundled corpus too small: %d", CommonCount())
	}
	all := CommonSeeds(0)
	if len(all) != CommonCount() {
		t.Fatalf("CommonSeeds(0) = %d, want %d", len(all), CommonCount())
	}
	// Most-common-first ordering: "123456" and "password" lead.
	if all[0] != "123456" {
		t.Errorf("first seed = %q, want 123456", all[0])
	}
	// Lowercased, no comments/blanks.
	for _, s := range all {
		if s == "" || strings.HasPrefix(s, "#") || s != strings.ToLower(s) {
			t.Errorf("bad seed %q", s)
		}
	}
	// top bound respected and returns a copy.
	top5 := CommonSeeds(5)
	if len(top5) != 5 {
		t.Fatalf("CommonSeeds(5) = %d", len(top5))
	}
	top5[0] = "x"
	if CommonSeeds(5)[0] == "x" {
		t.Error("CommonSeeds must return a copy")
	}
}

func TestBuildSeedCommon(t *testing.T) {
	cfg := &profile.Config{Profile: profile.Profile{FirstName: "John"}}
	cfg.Rules.SeedCommon = true
	cfg.Rules.SeedCommonTop = 10
	got := Build(cfg)
	contains(t, got, "john", "123456", "password", "qwerty")
	// Strictly sorted + unique.
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("not sorted/unique at %d", i)
		}
	}
}

func TestExtractCompanyFields(t *testing.T) {
	p := profile.Profile{
		Type:        "company",
		Company:     "Acme Corp",
		Brands:      []string{"Roadrunner"},
		Products:    []string{"Rocket Skates"},
		Locations:   []string{"Phoenix"},
		Departments: []string{"Engineering"},
		Founders:    []string{"Wile"},
		Founded:     1998,
	}
	got := Extract(p)
	contains(t, got, "acme", "corp", "roadrunner", "rocket", "skates", "phoenix", "engineering", "wile", "1998", "98")
}

func TestExtractChildren(t *testing.T) {
	got := Extract(profile.Profile{FirstName: "Pat", Children: []string{"Sam", "Max"}})
	contains(t, got, "pat", "sam", "max")
}
