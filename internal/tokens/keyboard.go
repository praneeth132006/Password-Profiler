// This file provides target-independent keyboard-walk seed tokens: the common
// adjacency patterns ("qwerty", "1q2w3e", "asdf") that appear at the top of
// every breach-composition study. They are opt-in (rules.keyboard_walks) so a
// targeted run stays targeted by default, and they flow through the same
// mutation/combination/policy pipeline as profile-derived tokens.
package tokens

// keyboardWalkSeeds is a curated, de-duplicated set of common keyboard walks
// for the standard US QWERTY layout, kept lowercase so the mutation stage owns
// case-explosion. Ordering is irrelevant; Build sorts the merged result.
var keyboardWalkSeeds = []string{
	// Horizontal rows, forwards.
	"qwerty", "qwertyuiop", "asdf", "asdfgh", "asdfghjkl", "zxcv", "zxcvbn", "zxcvbnm",
	// Horizontal rows, reversed.
	"ytrewq", "poiuytrewq", "fdsa", "lkjhgfdsa", "mnbvcxz",
	// Common partial / popular variants.
	"qwer", "wert", "erty", "qweasd", "qweasdzxc", "qazwsx", "qazwsxedc",
	"zaq", "zaq12wsx", "zaqwsx", "wsxedc", "edcrfv",
	// Vertical / diagonal columns.
	"qaz", "wsx", "edc", "rfv", "tgb", "yhn", "ujm", "plm", "okm", "ijn",
	// Digit-row walks and interleaved number/letter walks.
	"123qwe", "1qaz", "1qaz2wsx", "1q2w3e", "1q2w3e4r", "1q2w3e4r5t",
	"q1w2e3", "q1w2e3r4", "zaq1", "2wsx", "3edc",
	// Number-pad walks.
	"147", "159", "258", "369", "753", "951", "7410", "14789",
	// Adjacent-key classics that behave like walks.
	"asdfg", "sdfg", "dfgh", "fghj", "ghjk", "hjkl",
}

// KeyboardWalkSeeds returns a copy of the keyboard-walk seed tokens so callers
// cannot mutate the shared slice.
func KeyboardWalkSeeds() []string {
	out := make([]string, len(keyboardWalkSeeds))
	copy(out, keyboardWalkSeeds)
	return out
}
