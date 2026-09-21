package policy

import "testing"

func TestFilterActive(t *testing.T) {
	if New(Policy{}).Active() {
		t.Errorf("empty policy should be inactive")
	}
	if !New(Policy{MinLen: 8}).Active() {
		t.Errorf("min_len policy should be active")
	}
	if !New(Policy{Require: []string{"digit"}}).Active() {
		t.Errorf("require policy should be active")
	}
}

func TestInactiveAllowsEverything(t *testing.T) {
	f := New(Policy{})
	for _, s := range []string{"", "a", "P@ssw0rd", "1234567890123456789"} {
		if !f.Allow(s) {
			t.Errorf("inactive filter rejected %q", s)
		}
	}
}

func TestAllow(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		cand   string
		want   bool
	}{
		{"min length ok", Policy{MinLen: 8}, "password", true},
		{"min length short", Policy{MinLen: 8}, "pass", false},
		{"max length ok", Policy{MaxLen: 8}, "password", true},
		{"max length long", Policy{MaxLen: 8}, "passwords", false},
		{"range ok", Policy{MinLen: 6, MaxLen: 10}, "monkey1", true},
		{"require all classes met", Policy{Require: []string{"upper", "lower", "digit"}}, "Passw0rd", true},
		{"require upper missing", Policy{Require: []string{"upper"}}, "password1", false},
		{"require digit missing", Policy{Require: []string{"digit"}}, "Password", false},
		{"require special met", Policy{Require: []string{"special"}}, "pass!word", true},
		{"require special missing", Policy{Require: []string{"special"}}, "password1", false},
		{"combined len+classes ok", Policy{MinLen: 8, MaxLen: 16, Require: []string{"upper", "lower", "digit"}}, "Falcons2024", true},
		{"combined too short", Policy{MinLen: 12, Require: []string{"digit"}}, "Falcon1", false},
		{"unicode length counts runes", Policy{MaxLen: 3}, "naïve", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.policy).Allow(tt.cand); got != tt.want {
				t.Errorf("Allow(%q) = %v, want %v", tt.cand, got, tt.want)
			}
		})
	}
}

func TestIsSpecial(t *testing.T) {
	specials := "!@#$%^&*()_-+=.,"
	for _, r := range specials {
		if !isSpecial(r) {
			t.Errorf("%q should be special", r)
		}
	}
	for _, r := range "aZ0 " {
		if isSpecial(r) {
			t.Errorf("%q should NOT be special", r)
		}
	}
}
