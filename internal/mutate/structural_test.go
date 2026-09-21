package mutate

import (
	"reflect"
	"testing"
)

func TestStructuralPrimitives(t *testing.T) {
	if got := Reverse("abc"); got != "cba" {
		t.Errorf("Reverse = %q", got)
	}
	if got := Duplicate("ab"); got != "abab" {
		t.Errorf("Duplicate = %q", got)
	}
	if got := TruncateLast("abc"); got != "ab" {
		t.Errorf("TruncateLast = %q", got)
	}
	if got := TruncateLast("a"); got != "" {
		t.Errorf("TruncateLast single rune = %q, want empty", got)
	}
}

func TestStructuralVariants(t *testing.T) {
	got := structuralVariants("abc")
	want := []string{"cba", "abcabc", "ab"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("structuralVariants(abc) = %v, want %v", got, want)
	}
	// A palindrome-length-1 input: reverse == input (dropped), truncate empty (dropped).
	got = structuralVariants("a")
	want = []string{"aa"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("structuralVariants(a) = %v, want %v", got, want)
	}
}
