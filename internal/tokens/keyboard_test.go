package tokens

import (
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func TestBuildWithoutKeyboardWalks(t *testing.T) {
	cfg := &profile.Config{Profile: profile.Profile{FirstName: "John"}}
	got := Build(cfg)
	// Default: no keyboard-walk seeds leak in.
	absent(t, got, "qwerty", "1q2w3e", "qazwsx")
	contains(t, got, "john")
}

func TestBuildWithKeyboardWalks(t *testing.T) {
	cfg := &profile.Config{Profile: profile.Profile{FirstName: "John"}}
	cfg.Rules.KeyboardWalks = true
	got := Build(cfg)
	contains(t, got, "john", "qwerty", "1q2w3e", "qazwsx", "zxcvbnm")

	// Result stays sorted and unique.
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("Build output not strictly sorted/unique at %d: %q then %q", i, got[i-1], got[i])
		}
	}
}

func TestKeyboardWalkSeedsCopy(t *testing.T) {
	a := KeyboardWalkSeeds()
	if len(a) == 0 {
		t.Fatal("expected seeds")
	}
	a[0] = "MUTATED"
	if KeyboardWalkSeeds()[0] == "MUTATED" {
		t.Error("KeyboardWalkSeeds returned a reference to the shared slice")
	}
	// No duplicates in the curated set.
	seen := map[string]struct{}{}
	for _, s := range KeyboardWalkSeeds() {
		if _, dup := seen[s]; dup {
			t.Errorf("duplicate seed %q", s)
		}
		seen[s] = struct{}{}
	}
}
