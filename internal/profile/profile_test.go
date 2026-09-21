package profile

import (
	"strings"
	"testing"
)

const validYAML = `
profile:
  first_name: John
  last_name: Doe
  dob: 1990-05-12
  keywords: [chess]
rules:
  case: [lower, upper]
  leet: partial
  separators: ["", "."]
  affixes:
    years: {from_dob: true, extra: [2024]}
    suffixes: ["123"]
  max_combine: 2
  depth: 3
policy:
  min_len: 8
  max_len: 16
  require: [upper, lower, digit]
output:
  mode: wordlist
  budget: 100
  dedupe: true
`

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string // substring; "" means no error
		check   func(t *testing.T, c *Config)
	}{
		{
			name: "valid full config",
			yaml: validYAML,
			check: func(t *testing.T, c *Config) {
				if c.Profile.FirstName != "John" {
					t.Errorf("first_name = %q, want John", c.Profile.FirstName)
				}
				if c.Rules.Leet != "partial" {
					t.Errorf("leet = %q, want partial", c.Rules.Leet)
				}
				if c.Output.Budget != 100 {
					t.Errorf("budget = %d, want 100", c.Output.Budget)
				}
			},
		},
		{
			name: "defaults applied for minimal config",
			yaml: "profile:\n  keywords: [falcons]\n",
			check: func(t *testing.T, c *Config) {
				if c.Rules.Leet != "off" {
					t.Errorf("default leet = %q, want off", c.Rules.Leet)
				}
				if c.Output.Mode != "wordlist" {
					t.Errorf("default mode = %q, want wordlist", c.Output.Mode)
				}
				if c.Rules.MaxCombine != 1 || c.Rules.Depth != 1 {
					t.Errorf("default max_combine/depth = %d/%d, want 1/1", c.Rules.MaxCombine, c.Rules.Depth)
				}
				if c.Output.Budget != 5_000_000 {
					t.Errorf("default budget = %d, want 5000000", c.Output.Budget)
				}
			},
		},
		{
			name:    "empty profile rejected",
			yaml:    "rules:\n  leet: off\n",
			wantErr: "at least one field",
		},
		{
			name:    "unknown key rejected",
			yaml:    "profile:\n  first_name: John\n  bogus_field: x\n",
			wantErr: "parse config",
		},
		{
			name:    "bad date rejected",
			yaml:    "profile:\n  dob: 12-05-1990\n",
			wantErr: "YYYY-MM-DD",
		},
		{
			name:    "bad leet rejected",
			yaml:    "profile:\n  keywords: [x]\nrules:\n  leet: sometimes\n",
			wantErr: "off|partial|full",
		},
		{
			name:    "bad mode rejected",
			yaml:    "profile:\n  keywords: [x]\noutput:\n  mode: turbo\n",
			wantErr: "wordlist|rules",
		},
		{
			name:    "unknown case rejected",
			yaml:    "profile:\n  keywords: [x]\nrules:\n  case: [smallcaps]\n",
			wantErr: "unknown case transform",
		},
		{
			name:    "min>max rejected",
			yaml:    "profile:\n  keywords: [x]\npolicy:\n  min_len: 20\n  max_len: 8\n",
			wantErr: "min_len",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Parse([]byte(tt.yaml))
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
