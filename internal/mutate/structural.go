package mutate

// Reverse returns s with its runes in reverse order.
func Reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Duplicate returns s concatenated with itself.
func Duplicate(s string) string { return s + s }

// TruncateLast drops the final rune of s. Empty for single-rune input.
func TruncateLast(s string) string {
	r := []rune(s)
	if len(r) <= 1 {
		return ""
	}
	return string(r[:len(r)-1])
}

// structuralVariants returns the structural mutations of s (never s itself),
// in deterministic order: reverse, duplicate, truncate. Variants that would be
// empty or identical to s are dropped.
func structuralVariants(s string) []string {
	out := make([]string, 0, 3)
	if rev := Reverse(s); rev != s {
		out = append(out, rev)
	}
	out = append(out, Duplicate(s))
	if tr := TruncateLast(s); tr != "" {
		out = append(out, tr)
	}
	return out
}
