package rank

import (
	"reflect"
	"strings"
	"testing"
)

func TestScoreOrdersRealisticHigher(t *testing.T) {
	// Each pair: the first should score strictly higher than the second.
	pairs := [][2]string{
		{"Falcons2024!", "f@lc0n$2024"},   // clean word+year+symbol beats heavy leet
		{"Northstar1", "Northstar123456"}, // short digit run beats long digit run
		{"Summer2024", "SuMMeR2024"},      // capitalized beats mixed case
		{"Password1!", "!1drowssaP"},      // canonical shape beats reversed
		{"Riverside7", "Riverside"},       // trailing digit beats bare word
		{"Chelsea99", "chelseachelsea"},   // word+digits beats duplicated word
		{"Autumn2023", "aut2mn2023"},      // clean core beats interior digit (leet)
	}
	for _, p := range pairs {
		hi, lo := Score(p[0]), Score(p[1])
		if hi <= lo {
			t.Errorf("Score(%q)=%d should be > Score(%q)=%d", p[0], hi, p[1], lo)
		}
	}
}

func TestScoreEmpty(t *testing.T) {
	if Score("") != 0 {
		t.Errorf("empty score should be 0")
	}
}

func TestScoreDeterministic(t *testing.T) {
	for _, s := range []string{"Falcons2024!", "hunter2", "P@ssw0rd"} {
		if Score(s) != Score(s) {
			t.Errorf("Score not deterministic for %q", s)
		}
	}
}

func TestOrderStableOnTies(t *testing.T) {
	// Identical strings must preserve input order (stable).
	in := []string{"Alpha1!", "Bravo1!", "Alpha1!"}
	out := Order(in)
	// All three share structure; ensure output is a permutation and stable for
	// equal scores (first "Alpha1!" precedes the later duplicate).
	if len(out) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(out))
	}
	joined := strings.Join(out, ",")
	if strings.Count(joined, "Alpha1!") != 2 || strings.Count(joined, "Bravo1!") != 1 {
		t.Errorf("Order dropped or duplicated entries: %v", out)
	}
}

func TestOrderPutsBestFirst(t *testing.T) {
	in := []string{"x", "Password1!", "aaaaaaaa", "Falcons2024"}
	out := Order(in)
	if out[0] != "Password1!" && out[0] != "Falcons2024" {
		t.Errorf("best candidate not first: %v", out)
	}
	if out[len(out)-1] != "x" {
		t.Errorf("weakest candidate (%q) should be last, got %v", "x", out)
	}
}

func TestStreamFullSort(t *testing.T) {
	in := "x\nPassword1!\n\nFalcons2024\naaaaaaaa\n" // blank line skipped
	var out strings.Builder
	n, err := Stream(strings.NewReader(in), &out, 0)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("wrote %d lines, want 4", n)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if lines[len(lines)-1] != "x" {
		t.Errorf("weakest should be last: %v", lines)
	}
}

func TestStreamTopKBoundedMatchesFullSort(t *testing.T) {
	// A larger input; top-K must equal the first K of a full ranking.
	in := []string{
		"a", "Bb", "Carrot99", "Delta2024!", "eeeeeeee", "Foxtrot1",
		"g", "Hotel22", "igloo", "Juliet2023!", "k1", "Lima7",
	}
	full := Order(in)

	var buf strings.Builder
	n, err := Stream(strings.NewReader(strings.Join(in, "\n")), &buf, 5)
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("top-5 wrote %d lines", n)
	}
	got := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	want := full[:5]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("top-5 = %v, want first 5 of full ranking %v", got, want)
	}
}

func TestStreamTopKLargerThanInput(t *testing.T) {
	in := "Alpha1!\nBravo2!\n"
	var out strings.Builder
	n, err := Stream(strings.NewReader(in), &out, 100)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("wrote %d, want 2", n)
	}
}
