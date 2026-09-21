package mutate

import "testing"

func TestCaseFuncs(t *testing.T) {
	tests := []struct {
		name string
		fn   CaseFunc
		in   string
		want string
	}{
		{"identity", Identity, "jOhn", "jOhn"},
		{"lower", Lower, "JoHN", "john"},
		{"upper", Upper, "john", "JOHN"},
		{"capitalize", Capitalize, "jOHN", "John"},
		{"capitalize empty", Capitalize, "", ""},
		{"toggle keeps tail", Toggle, "jOHN", "JOHN"},
		{"toggle simple", Toggle, "john", "John"},
		{"invert", Invert, "John", "jOHN"},
		{"invert digits untouched", Invert, "aB3", "Ab3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(tt.in); got != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
			}
		})
	}
}
