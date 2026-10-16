// This file exposes a bundled, curated common-password corpus — the universally
// known weak passwords that top every breach-composition study. It is public
// composition knowledge (not anyone's leaked credentials) and is used only as
// an OPT-IN seed / known-weak signal for authorized audits.
//
// Use it to (a) seed generation with high-prevalence passwords so a run always
// tries the obvious guesses, and (b) as a starting corpus you can supplement
// with a larger external list via the CLI. For big external corpora (SecLists,
// rockyou, HIBP), prefer `rank --model` (ordering) and `rank --exclude`
// (skip-known) rather than injecting millions of tokens into generation.
package tokens

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed data/common.txt
var commonRaw string

// commonList is the parsed, order-preserving (most-common-first) corpus.
var commonList = sync.OnceValue(func() []string {
	var out []string
	for _, line := range strings.Split(commonRaw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, strings.ToLower(line))
	}
	return out
})

// CommonSeeds returns up to top entries from the bundled common-password corpus,
// most common first. top <= 0 or top beyond the corpus returns the whole list.
// The result is a fresh slice the caller may retain or mutate.
func CommonSeeds(top int) []string {
	src := commonList()
	if top <= 0 || top > len(src) {
		top = len(src)
	}
	out := make([]string, top)
	copy(out, src[:top])
	return out
}

// CommonCount reports how many entries the bundled corpus holds.
func CommonCount() int { return len(commonList()) }
