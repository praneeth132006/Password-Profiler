package rules

import (
	"bytes"
	"strings"
	"testing"
)

func TestApply(t *testing.T) {
	tests := []struct {
		rule string
		word string
		want string
	}{
		{":", "john", "john"},
		{"l", "JOHN", "john"},
		{"u", "john", "JOHN"},
		{"c", "jOHN", "John"},
		{"C", "john", "jOHN"},
		{"t", "John", "jOHN"},
		{"T0", "john", "John"},
		{"r", "john", "nhoj"},
		{"d", "ab", "abab"},
		{"]", "john", "joh"},
		{"[", "john", "ohn"},
		{"$1$2$3", "john", "john123"},
		{"$!", "john", "john!"},
		{"^0^9^9^1", "john", "1990john"}, // prepend "1990"
		{"so0", "john", "j0hn"},
		{"sa@se3", "cafe", "c@f3"},
	}
	for _, tt := range tests {
		t.Run(tt.rule+"/"+tt.word, func(t *testing.T) {
			got, err := Apply(tt.word, tt.rule)
			if err != nil {
				t.Fatalf("Apply(%q,%q) error: %v", tt.word, tt.rule, err)
			}
			if got != tt.want {
				t.Errorf("Apply(%q,%q) = %q, want %q", tt.word, tt.rule, got, tt.want)
			}
		})
	}
}

func TestApplyRejectsUnknown(t *testing.T) {
	if _, err := Apply("john", "Z"); err == nil {
		t.Errorf("expected error for unknown function")
	}
	if _, err := Apply("john", "$"); err == nil {
		t.Errorf("expected error for $ without argument")
	}
}

func TestGenerateContents(t *testing.T) {
	set := Generate(Config{
		Cases:      []string{"lower", "capitalize", "upper"},
		Leet:       "partial",
		Append:     []string{"123", "!"},
		Prepend:    []string{"1990"},
		Structural: true,
	})
	lines := set.Lines()

	// No-op must be first.
	if lines[0] != ":" {
		t.Errorf("first rule = %q, want :", lines[0])
	}
	want := []string{"l", "c", "u", "sa@", "so0", "$1$2$3", "$!", "^0^9^9^1", "r", "d", "]"}
	for _, w := range want {
		if !hasLine(lines, w) {
			t.Errorf("missing rule %q in %v", w, lines)
		}
	}
	// Deterministic + unique.
	seen := map[string]bool{}
	for _, l := range lines {
		if seen[l] {
			t.Errorf("duplicate rule line %q", l)
		}
		seen[l] = true
	}
}

// TestGeneratedRulesProduceIntendedCandidates is the key correctness proof:
// every generated rule must parse and, applied to a base word, yield exactly
// the mutation it encodes.
func TestGeneratedRulesProduceIntendedCandidates(t *testing.T) {
	set := Generate(Config{
		Cases:      []string{"lower", "capitalize", "upper"},
		Leet:       "partial",
		Append:     []string{"123", "2024"},
		Prepend:    []string{"1990"},
		Structural: true,
	})
	base := "john"
	// Every rule must apply without error.
	for _, rule := range set.Lines() {
		if _, err := Apply(base, rule); err != nil {
			t.Fatalf("generated rule %q failed to apply: %v", rule, err)
		}
	}
	// Spot-check specific intended candidates are reachable.
	intended := map[string]string{
		"c":        "John",     // capitalize
		"u":        "JOHN",     // upper
		"so0":      "j0hn",     // leet o->0
		"$1$2$3":   "john123",  // append 123
		"$2$0$2$4": "john2024", // append 2024
		"^0^9^9^1": "1990john", // prepend 1990
		"r":        "nhoj",     // reverse
		"d":        "johnjohn", // duplicate
		"]":        "joh",      // truncate
	}
	for rule, wantCand := range intended {
		if !hasLine(set.Lines(), rule) {
			t.Errorf("expected generated rule %q (for candidate %q)", rule, wantCand)
			continue
		}
		got, err := Apply(base, rule)
		if err != nil {
			t.Fatalf("Apply(%q,%q): %v", base, rule, err)
		}
		if got != wantCand {
			t.Errorf("rule %q on %q = %q, want %q", rule, base, got, wantCand)
		}
	}
}

func TestSetWriteTo(t *testing.T) {
	set := Generate(Config{Cases: []string{"lower"}})
	var buf bytes.Buffer
	if _, err := set.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, ":\n") {
		t.Errorf("output should start with the no-op rule, got %q", out)
	}
	if strings.Count(out, "\n") != set.Len() {
		t.Errorf("line count %d != Len %d", strings.Count(out, "\n"), set.Len())
	}
}

func hasLine(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

func TestWriteJohnSection(t *testing.T) {
	set := Generate(Config{Cases: []string{"lower", "capitalize"}, Append: []string{"1"}})
	var buf bytes.Buffer
	if _, err := set.WriteJohn(&buf, "audit"); err != nil {
		t.Fatalf("WriteJohn: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "[List.Rules:audit]\n") {
		t.Errorf("missing John section header: %q", out)
	}
	// Every hashcat line must also appear in the John output (shared syntax).
	for _, line := range set.Lines() {
		if !strings.Contains(out, "\n"+line+"\n") {
			t.Errorf("John output missing rule %q", line)
		}
	}
	// Default section name when empty.
	var buf2 bytes.Buffer
	_, _ = set.WriteJohn(&buf2, "")
	if !strings.HasPrefix(buf2.String(), "[List.Rules:pwprofiler]\n") {
		t.Errorf("empty section should default to pwprofiler")
	}
}
