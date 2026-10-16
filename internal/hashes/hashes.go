// Package hashes computes the hash of a candidate password for the fast,
// unsalted algorithms used in authorized offline password audits, so an auditor
// can match a captured hash against a generated candidate list.
//
// AUTHORIZED TESTING ONLY. These are deterministic digests of candidate guesses
// the auditor already holds; they contain no secret material. Salted/slow
// schemes (bcrypt, argon2, scrypt, yescrypt, PBKDF2) are intentionally NOT here:
// their output depends on a per-hash salt, so a precomputed candidate:hash list
// is useless for matching — crack those directly with hashcat/John and a
// wordlist instead.
package hashes

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
)

// Algo identifies a supported hash algorithm.
type Algo struct {
	Name        string // canonical lower-case name used on the CLI
	HashcatMode int    // hashcat -m value (-1 if not applicable)
	JohnFormat  string // John the Ripper --format name
	sum         func([]byte) string
}

// Hash returns the lower-case hex digest of pw under this algorithm.
func (a Algo) Hash(pw string) string { return a.sum([]byte(pw)) }

func hexHash(b []byte) string { return hex.EncodeToString(b) }

// ntlm is MD4 of the password encoded as UTF-16LE — the NT hash used by Windows
// / Active Directory. It is the single most useful fast hash in enterprise audits.
func ntlm(pw []byte) string {
	u := utf16.Encode([]rune(string(pw)))
	buf := make([]byte, len(u)*2)
	for i, r := range u {
		buf[i*2] = byte(r)
		buf[i*2+1] = byte(r >> 8)
	}
	sum := md4Sum(buf)
	return hexHash(sum[:])
}

// registry holds every supported algorithm keyed by canonical name.
var registry = map[string]Algo{
	"md5":    {Name: "md5", HashcatMode: 0, JohnFormat: "raw-md5", sum: func(b []byte) string { h := md5.Sum(b); return hexHash(h[:]) }},
	"sha1":   {Name: "sha1", HashcatMode: 100, JohnFormat: "raw-sha1", sum: func(b []byte) string { h := sha1.Sum(b); return hexHash(h[:]) }},
	"sha256": {Name: "sha256", HashcatMode: 1400, JohnFormat: "raw-sha256", sum: func(b []byte) string { h := sha256.Sum256(b); return hexHash(h[:]) }},
	"sha512": {Name: "sha512", HashcatMode: 1700, JohnFormat: "raw-sha512", sum: func(b []byte) string { h := sha512.Sum512(b); return hexHash(h[:]) }},
	"ntlm":   {Name: "ntlm", HashcatMode: 1000, JohnFormat: "nt", sum: func(b []byte) string { return ntlm(b) }},
}

// Supported returns the canonical algorithm names in a stable, documented order.
func Supported() []string { return []string{"md5", "sha1", "sha256", "sha512", "ntlm"} }

// Lookup resolves a case-insensitive algorithm name. Common aliases are accepted
// (e.g. "sha-1", "nt", "md-5").
func Lookup(name string) (Algo, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.ReplaceAll(key, "-", "")
	switch key {
	case "nt":
		key = "ntlm"
	case "sha":
		key = "sha1"
	}
	a, ok := registry[key]
	return a, ok
}

// Set is an ordered, de-duplicated selection of algorithms. The order follows
// Supported() so output columns are stable regardless of request order.
type Set struct{ algos []Algo }

// NewSet resolves a list of algorithm names into a Set, rejecting unknown names.
// An empty input yields an empty Set (no error).
func NewSet(names []string) (Set, error) {
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		for _, part := range strings.Split(n, ",") { // allow comma-joined values
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			a, ok := Lookup(part)
			if !ok {
				return Set{}, fmt.Errorf("unknown hash algorithm %q (supported: %s)", part, strings.Join(Supported(), ", "))
			}
			seen[a.Name] = struct{}{}
		}
	}
	var out []Algo
	for _, name := range Supported() {
		if _, ok := seen[name]; ok {
			out = append(out, registry[name])
		}
	}
	return Set{algos: out}, nil
}

// Empty reports whether the set selects no algorithms.
func (s Set) Empty() bool { return len(s.algos) == 0 }

// Names returns the canonical names of the selected algorithms, in column order.
func (s Set) Names() []string {
	out := make([]string, len(s.algos))
	for i, a := range s.algos {
		out[i] = a.Name
	}
	return out
}

// Algos returns the selected algorithms in column order.
func (s Set) Algos() []Algo {
	out := make([]Algo, len(s.algos))
	copy(out, s.algos)
	return out
}

// Line formats a candidate and its hashes for output using sep between fields:
//
//	pw<sep>hash1<sep>hash2...
//
// With an empty Set it returns pw unchanged.
func (s Set) Line(pw, sep string) string {
	if len(s.algos) == 0 {
		return pw
	}
	var b strings.Builder
	b.WriteString(pw)
	for _, a := range s.algos {
		b.WriteString(sep)
		b.WriteString(a.Hash(pw))
	}
	return b.String()
}
