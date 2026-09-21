package dedup

import (
	"fmt"
	"testing"
)

// deduperContract runs the behavioral contract both implementations must honor:
// a first sighting is new, an immediate repeat is seen, and no previously-added
// item is ever reported as new (Bloom may over-report, never under-report).
func deduperContract(t *testing.T, d Deduper) {
	t.Helper()
	if d.Seen("alpha") {
		t.Errorf("first Seen(alpha) = true, want false")
	}
	if !d.Seen("alpha") {
		t.Errorf("repeat Seen(alpha) = false, want true")
	}
	if d.Seen("beta") {
		t.Errorf("first Seen(beta) = true, want false")
	}
	// Everything added so far must still read as seen.
	for _, s := range []string{"alpha", "beta"} {
		if !d.Seen(s) {
			t.Errorf("Seen(%q) = false after add, want true", s)
		}
	}
}

func TestExactContract(t *testing.T) { deduperContract(t, NewExact()) }
func TestBloomContract(t *testing.T) { deduperContract(t, NewBloom(1000, 0.001)) }

func TestExactIsPrecise(t *testing.T) {
	e := NewExact()
	for i := 0; i < 10000; i++ {
		if e.Seen(fmt.Sprintf("item-%d", i)) {
			t.Fatalf("exact deduper falsely reported item-%d as seen", i)
		}
	}
}

func TestExactKindBloomKind(t *testing.T) {
	if NewExact().Kind() != "exact" {
		t.Errorf("Exact.Kind mismatch")
	}
	if NewBloom(10, 0.01).Kind() != "bloom" {
		t.Errorf("Bloom.Kind mismatch")
	}
}

// TestBloomFalsePositiveRate loads the filter to its design capacity, then
// measures the false-positive rate on fresh items using a read-only probe (so
// the measurement itself does not add load). It also asserts there are no false
// negatives: every inserted item must still probe as present.
func TestBloomFalsePositiveRate(t *testing.T) {
	const n = 20000
	const target = 0.01
	b := NewBloom(n, target)

	for i := 0; i < n; i++ {
		b.Seen(fmt.Sprintf("known-%d", i))
	}
	// No false negatives.
	for i := 0; i < n; i++ {
		if !b.probe(fmt.Sprintf("known-%d", i)) {
			t.Fatalf("false negative: known-%d not present", i)
		}
	}
	// Measure false positives on never-inserted items (read-only).
	falsePos := 0
	const probes = 50000
	for i := 0; i < probes; i++ {
		if b.probe(fmt.Sprintf("fresh-%d", i)) {
			falsePos++
		}
	}
	rate := float64(falsePos) / float64(probes)
	// At design load the rate should be near the 1% target; allow slack.
	if rate > 0.02 {
		t.Errorf("false-positive rate %.4f exceeds bound 0.02 (target %.3f)", rate, target)
	}
	t.Logf("measured false-positive rate at capacity: %.4f (target %.3f)", rate, target)
}
