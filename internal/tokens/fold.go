// This file adds transliteration / diacritic folding to token extraction.
// Targets routinely register accented names ("José", "Müller") but type ASCII
// passwords, so for every accented word we also emit its folded ASCII form(s).
// Everything here is self-contained (no external deps) and deterministic.
package tokens

import "strings"

// simpleFold maps a single accented rune to its closest ASCII base letter.
// Covers Latin-1 Supplement and the common Latin Extended-A letters seen in
// European names. Runes not in the table are passed through unchanged.
var simpleFold = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c", 'ĉ': "c", 'ċ': "c",
	'ď': "d", 'đ': "d", 'ð': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ĝ': "g", 'ğ': "g", 'ġ': "g", 'ģ': "g",
	'ĥ': "h", 'ħ': "h",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'ĭ': "i", 'į': "i", 'ı': "i",
	'ĵ': "j",
	'ķ': "k",
	'ĺ': "l", 'ļ': "l", 'ľ': "l", 'ł': "l",
	'ñ': "n", 'ń': "n", 'ņ': "n", 'ň': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ŏ': "o", 'ő': "o",
	'ŕ': "r", 'ŗ': "r", 'ř': "r",
	'ś': "s", 'ŝ': "s", 'ş': "s", 'š': "s",
	'ţ': "t", 'ť': "t", 'ŧ': "t",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'ŵ': "w",
	'ý': "y", 'ÿ': "y", 'ŷ': "y",
	'ź': "z", 'ż': "z", 'ž': "z",
	'þ': "th",
}

// digraphFold maps runes whose conventional romanization is a multi-letter
// digraph (German/Nordic style). These produce a *second* folded variant so a
// run covers both "muller" and "mueller", "strasse" etc.
var digraphFold = map[rune]string{
	'ä': "ae", 'ö': "oe", 'ü': "ue",
	'ß': "ss", 'æ': "ae", 'œ': "oe",
	'å': "aa",
}

// foldSimple returns s with every rune mapped through simpleFold (ß/æ/œ, which
// have no single-letter base, expand to their digraph here so nothing is lost).
func foldSimple(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case r == 'ß':
			b.WriteString("ss")
		case r == 'æ':
			b.WriteString("ae")
		case r == 'œ':
			b.WriteString("oe")
		default:
			if v, ok := simpleFold[r]; ok {
				b.WriteString(v)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// foldDigraph returns s with digraph-preferring runes expanded (ä->ae) and the
// remaining accented runes folded to their simple base.
func foldDigraph(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		default:
			if v, ok := digraphFold[r]; ok {
				b.WriteString(v)
			} else if v, ok := simpleFold[r]; ok {
				b.WriteString(v)
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// hasNonASCII reports whether s contains any byte >= 0x80, i.e. whether folding
// could possibly change it. Pure-ASCII words skip the folding work entirely.
func hasNonASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return true
		}
	}
	return false
}

// foldVariants returns the ASCII-folded forms of a word that differ from the
// word itself: the simple fold always, plus the digraph fold when the word
// contains a digraph-preferring rune and the two folds diverge. The input is
// assumed already lowercased. Pure-ASCII input returns nil.
func foldVariants(word string) []string {
	if !hasNonASCII(word) {
		return nil
	}
	var out []string
	simple := foldSimple(word)
	if simple != word && simple != "" {
		out = append(out, simple)
	}
	digraph := foldDigraph(word)
	if digraph != word && digraph != "" && digraph != simple {
		out = append(out, digraph)
	}
	return out
}
