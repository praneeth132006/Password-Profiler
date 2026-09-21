// Package dedup provides candidate de-duplication strategies for the output
// stage. Two implementations trade memory for exactness:
//
//   - Exact: a hash set. Correct, but stores every unique candidate, so RAM
//     grows with output size — fine for small runs, ruinous for billions.
//   - Bloom: a Bloom filter. Memory is fixed up front from the expected item
//     count and a target false-positive rate, independent of how many
//     candidates actually flow through. The cost is a small, bounded chance of
//     dropping a genuinely-new candidate (a false positive reported as a
//     duplicate). For wordlist generation that trade is almost always worth it:
//     ~9 MB holds 5M items at a 0.1% false-positive rate, versus hundreds of MB
//     for an exact set.
package dedup

import (
	"hash/fnv"
	"math"
)

// Deduper records candidates and reports whether one has been seen before.
type Deduper interface {
	// Seen returns true if s has (probably) been seen already; otherwise it
	// records s and returns false. Implementations may over-report (Bloom) but
	// never under-report a previously-seen item.
	Seen(s string) bool
	// Kind is a short label for stats output ("exact", "bloom").
	Kind() string
}

// Exact is a hash-set deduper: precise, unbounded memory.
type Exact struct {
	seen map[string]struct{}
}

// NewExact returns an exact deduper.
func NewExact() *Exact {
	return &Exact{seen: make(map[string]struct{})}
}

// Seen implements Deduper.
func (e *Exact) Seen(s string) bool {
	if _, ok := e.seen[s]; ok {
		return true
	}
	e.seen[s] = struct{}{}
	return false
}

// Kind implements Deduper.
func (e *Exact) Kind() string { return "exact" }

// Bloom is a fixed-memory probabilistic deduper.
type Bloom struct {
	bits []uint64
	m    uint64 // number of bits
	k    uint64 // number of hash functions
}

// NewBloom sizes a Bloom filter for expectedItems at target false-positive rate
// fpRate (e.g. 0.001 for 0.1%). Both are clamped to sane minimums.
func NewBloom(expectedItems int, fpRate float64) *Bloom {
	n := float64(expectedItems)
	if n < 1 {
		n = 1
	}
	if fpRate <= 0 || fpRate >= 1 {
		fpRate = 0.001
	}
	// Optimal m (bits) and k (hashes) for n items at fpRate.
	ln2 := math.Ln2
	m := math.Ceil(-(n * math.Log(fpRate)) / (ln2 * ln2))
	k := math.Round((m / n) * ln2)
	if m < 64 {
		m = 64
	}
	if k < 1 {
		k = 1
	}
	words := (uint64(m) + 63) / 64
	return &Bloom{
		bits: make([]uint64, words),
		m:    words * 64,
		k:    uint64(k),
	}
}

// Seen implements Deduper. It tests-and-sets the k bit positions for s.
func (b *Bloom) Seen(s string) bool {
	h1, h2 := hashes(s)
	all := true
	for i := uint64(0); i < b.k; i++ {
		pos := b.position(h1, h2, i)
		word, mask := pos/64, uint64(1)<<(pos%64)
		if b.bits[word]&mask == 0 {
			all = false
			b.bits[word] |= mask
		}
	}
	return all
}

// probe reports whether s's bits are all set, without modifying the filter.
// Used for read-only measurement (e.g. false-positive-rate tests).
func (b *Bloom) probe(s string) bool {
	h1, h2 := hashes(s)
	for i := uint64(0); i < b.k; i++ {
		pos := b.position(h1, h2, i)
		if b.bits[pos/64]&(uint64(1)<<(pos%64)) == 0 {
			return false
		}
	}
	return true
}

// position derives the i-th bit index via Kirsch–Mitzenmacher double hashing
// (with an i^2 term to avoid short cycles).
func (b *Bloom) position(h1, h2, i uint64) uint64 {
	return (h1 + i*h2 + i*i) % b.m
}

// Kind implements Deduper.
func (b *Bloom) Kind() string { return "bloom" }

// hashes returns two well-mixed, near-independent 64-bit hashes of s for double
// hashing. A single FNV-1a hash seeds two splitmix64 streams; splitmix64's
// strong avalanche decorrelates the two outputs, keeping the Bloom filter's
// observed false-positive rate close to the theoretical target.
func hashes(s string) (uint64, uint64) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	base := h.Sum64()

	h1 := splitmix64(base)
	h2 := splitmix64(base + 0x9e3779b97f4a7c15)
	if h2 == 0 {
		h2 = 0x9e3779b97f4a7c15 // ensure a non-zero step
	}
	return h1, h2
}

// splitmix64 is a fast, high-quality 64-bit mixer.
func splitmix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}
