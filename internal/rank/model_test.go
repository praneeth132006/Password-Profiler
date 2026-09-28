package rank

import (
	"strings"
	"testing"
)

func TestSuffixBonusBreaksTies(t *testing.T) {
	// Same shape, different tail: the common tail must rank higher.
	if Score("Summer1") <= Score("Summer4817") {
		t.Errorf("common suffix 1 should outrank rare tail 4817")
	}
	if suffixBonus("Summerxyz") != 0 {
		t.Errorf("no numeric/symbol tail => no bonus")
	}
}

func TestTrailingTail(t *testing.T) {
	cases := map[string]string{
		"Falcons2024!": "2024!",
		"hunter2":      "2",
		"password":     "",
		"abc123":       "123",
	}
	for in, want := range cases {
		if got := trailingTail(in); got != want {
			t.Errorf("trailingTail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestModelBoostsCorpusTerms(t *testing.T) {
	// A frequency-ordered corpus: "acme" most frequent, then "falcons".
	m, err := LoadModel(strings.NewReader("# corpus\nacme\nfalcons\nchess\n"))
	if err != nil {
		t.Fatal(err)
	}
	base := Score("Acme2024!")
	boosted := m.Score("Acme2024!")
	if boosted <= base {
		t.Errorf("model should boost a corpus-word core: base=%d boosted=%d", base, boosted)
	}
	// A candidate whose core is NOT in the corpus gets no boost.
	if m.Score("Zzzzz2024!") != Score("Zzzzz2024!") {
		t.Errorf("non-corpus core should not be boosted")
	}
	// Higher-frequency corpus term boosts at least as much as a lower one.
	if m.bonus("acme") < m.bonus("chess") {
		t.Errorf("more frequent term should not boost less")
	}
}

func TestModelEmpty(t *testing.T) {
	m, err := LoadModel(strings.NewReader("\n#only comments\n"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Score("Acme1") != Score("Acme1") {
		t.Errorf("empty model must be a no-op")
	}
}

func TestStreamWithExclude(t *testing.T) {
	exclude := map[string]struct{}{"Northstar1": {}, "aaaaaaaa": {}}
	var out strings.Builder
	n, err := StreamWith(strings.NewReader("Northstar1\nFalcons2024!\naaaaaaaa\nSummer2023\n"),
		&out, Options{Exclude: exclude})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 remaining after exclusion, got %d", n)
	}
	if strings.Contains(out.String(), "Northstar1") || strings.Contains(out.String(), "aaaaaaaa") {
		t.Errorf("excluded candidates leaked into output: %q", out.String())
	}
}

func TestStreamWithModelOrders(t *testing.T) {
	m, _ := LoadModel(strings.NewReader("chess\n"))
	var out strings.Builder
	// Without the model, "Falcons2024!" would lead; the corpus should lift Chess.
	_, err := StreamWith(strings.NewReader("Falcons2024!\nChess2024!\n"), &out, Options{Model: m})
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(strings.TrimSpace(out.String()), "\n", 2)[0]
	if first != "Chess2024!" {
		t.Errorf("model corpus should rank Chess2024! first, got %q", first)
	}
}
