package mutate

import (
	"reflect"
	"testing"
)

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

func TestEngineExpand(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		in   string
		want []string
	}{
		{
			name: "case only",
			cfg:  Config{Cases: []string{"lower", "capitalize", "upper"}},
			in:   "john",
			want: []string{"john", "John", "JOHN"},
		},
		{
			name: "case plus suffixes",
			cfg:  Config{Cases: []string{"lower", "capitalize"}, Suffixes: []string{"123", "!"}},
			in:   "rex",
			want: []string{"rex", "rex123", "rex!", "Rex", "Rex123", "Rex!"},
		},
		{
			name: "empty suffix ignored, dedup across cases",
			cfg:  Config{Cases: []string{"lower", "identity"}, Suffixes: []string{"", "1"}},
			in:   "abc", // lower(abc)=abc, identity(abc)=abc -> deduped
			want: []string{"abc", "abc1"},
		},
		{
			name: "no configured cases falls back to identity",
			cfg:  Config{Cases: nil, Suffixes: []string{"9"}},
			in:   "kite",
			want: []string{"kite", "kite9"},
		},
		{
			name: "empty token yields nothing",
			cfg:  Config{Cases: []string{"lower"}},
			in:   "",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(tt.cfg)
			got := eng.Expand(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expand(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
