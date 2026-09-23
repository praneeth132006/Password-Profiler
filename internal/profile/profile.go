// Package profile parses and validates the YAML configuration that drives the
// pwprofiler pipeline. It defines the full config schema up front so that a
// sample config parses cleanly, while later phases progressively consume more
// of the surface (leet, combination, policy, rules).
package profile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root of the YAML document.
type Config struct {
	Profile Profile `yaml:"profile"`
	Rules   Rules   `yaml:"rules"`
	Policy  Policy  `yaml:"policy"`
	Output  Output  `yaml:"output"`
}

// Profile holds the OSINT collected about a target. Every field is optional on
// its own, but validation requires at least one usable value overall.
type Profile struct {
	FirstName string   `yaml:"first_name"`
	LastName  string   `yaml:"last_name"`
	Nickname  string   `yaml:"nickname"`
	DOB       string   `yaml:"dob"` // YYYY-MM-DD
	Partner   string   `yaml:"partner"`
	Pets      []string `yaml:"pets"`
	Company   string   `yaml:"company"`
	Domain    string   `yaml:"domain"`
	Keywords  []string `yaml:"keywords"`
}

// Rules configures the mutation and combination stages.
type Rules struct {
	Exhaustive        bool     `yaml:"exhaustive"`
	RepeatTokens      bool     `yaml:"repeat_tokens"`
	LeetCap           int      `yaml:"leet_cap"`
	MaxVariants       int      `yaml:"max_variants"`
	MaxAttempts       int      `yaml:"max_attempts"`
	MaxCandidateBytes int      `yaml:"max_candidate_bytes"`
	CombineLimit      int      `yaml:"combine_limit"`
	CombineAttempts   int      `yaml:"combine_attempts"`
	Case              []string `yaml:"case"`       // lower | upper | capitalize | toggle | invert | identity
	Leet              string   `yaml:"leet"`       // off | partial | full
	Separators        []string `yaml:"separators"` // e.g. "", ".", "_"
	Affixes           Affixes  `yaml:"affixes"`
	MaxCombine        int      `yaml:"max_combine"` // max tokens joined in one candidate
	Depth             int      `yaml:"depth"`       // max chained mutations
}

// Affixes configures prepend/append material.
type Affixes struct {
	Years    Years    `yaml:"years"`
	Suffixes []string `yaml:"suffixes"`
}

// Years derives year affixes from the target's DOB and/or explicit extras.
type Years struct {
	FromDOB bool  `yaml:"from_dob"`
	Extra   []int `yaml:"extra"`
}

// Policy is the target password policy candidates must satisfy (Phase 5).
type Policy struct {
	MaxBytes  int      `yaml:"max_bytes"`
	MaxRepeat int      `yaml:"max_repeat"`
	Forbidden string   `yaml:"forbidden"`
	Blocklist []string `yaml:"blocklist"`
	MinLen    int      `yaml:"min_len"`
	MaxLen    int      `yaml:"max_len"`
	Require   []string `yaml:"require"` // upper | lower | digit | special
}

// Output controls where and how candidates are emitted.
type Output struct {
	Mode   string `yaml:"mode"` // wordlist | rules
	File   string `yaml:"file"`
	Budget int    `yaml:"budget"`
	Dedupe bool   `yaml:"dedupe"`
}

// Load reads, parses, and validates a config file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	return Parse(raw)
}

// Parse validates YAML bytes into a Config. It is separated from Load so tests
// can exercise it without touching the filesystem.
func Parse(raw []byte) (*Config, error) {
	var cfg Config
	// Reject unknown keys so typos surface instead of being silently ignored.
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("config must contain exactly one YAML document")
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// applyDefaults fills in safe defaults for omitted fields.
func (c *Config) applyDefaults() {
	if c.Rules.MaxVariants == 0 {
		c.Rules.MaxVariants = 10000
	}
	if c.Rules.MaxAttempts == 0 {
		c.Rules.MaxAttempts = 100000
	}
	if c.Rules.MaxCandidateBytes == 0 {
		c.Rules.MaxCandidateBytes = 4096
	}
	if c.Rules.CombineLimit == 0 {
		c.Rules.CombineLimit = 100000
	}
	if c.Rules.CombineAttempts == 0 {
		c.Rules.CombineAttempts = 1000000
	}
	if c.Rules.Leet == "" {
		c.Rules.Leet = "off"
	}
	if len(c.Rules.Case) == 0 {
		c.Rules.Case = []string{"identity"}
	}
	if len(c.Rules.Separators) == 0 {
		c.Rules.Separators = []string{""}
	}
	if c.Rules.MaxCombine == 0 {
		c.Rules.MaxCombine = 1
	}
	if c.Rules.Depth == 0 {
		c.Rules.Depth = 1
	}
	if c.Output.Mode == "" {
		c.Output.Mode = "wordlist"
	}
	if c.Output.Budget == 0 {
		c.Output.Budget = 5_000_000
	}
}

func (c *Config) validate() error {
	if c.Rules.LeetCap < -1 {
		return fmt.Errorf("rules.leet_cap must be -1 (all positions), 0 (default 3), or positive")
	}
	if c.Rules.Exhaustive && c.Output.Mode != "wordlist" {
		return fmt.Errorf("exhaustive mode requires wordlist output")
	}
	for _, limit := range []struct {
		name  string
		value int
	}{{"max_variants", c.Rules.MaxVariants}, {"max_attempts", c.Rules.MaxAttempts}, {"max_candidate_bytes", c.Rules.MaxCandidateBytes}, {"combine_limit", c.Rules.CombineLimit}, {"combine_attempts", c.Rules.CombineAttempts}} {
		if limit.value < 1 {
			return fmt.Errorf("rules.%s must be positive", limit.name)
		}
	}
	if c.Rules.MaxCombine > 8 {
		return fmt.Errorf("rules.max_combine must be between 1 and 8")
	}
	if c.Rules.Depth > 8 {
		return fmt.Errorf("rules.depth must be between 1 and 8")
	}
	if c.Output.Budget < 0 {
		return fmt.Errorf("output.budget must be positive")
	}
	if c.Profile.isEmpty() {
		return fmt.Errorf("profile: at least one field (name, keywords, company, ...) is required")
	}
	if c.Profile.DOB != "" {
		if _, err := time.Parse("2006-01-02", c.Profile.DOB); err != nil {
			return fmt.Errorf("profile.dob %q: must be YYYY-MM-DD: %w", c.Profile.DOB, err)
		}
	}
	switch c.Rules.Leet {
	case "off", "partial", "full":
	default:
		return fmt.Errorf("rules.leet %q: must be off|partial|full", c.Rules.Leet)
	}
	for _, cs := range c.Rules.Case {
		switch cs {
		case "identity", "lower", "upper", "capitalize", "toggle", "invert":
		default:
			return fmt.Errorf("rules.case %q: unknown case transform", cs)
		}
	}
	switch c.Output.Mode {
	case "wordlist", "rules":
	default:
		return fmt.Errorf("output.mode %q: must be wordlist|rules", c.Output.Mode)
	}
	if c.Rules.MaxCombine < 1 {
		return fmt.Errorf("rules.max_combine %d: must be >= 1", c.Rules.MaxCombine)
	}
	if c.Rules.Depth < 1 {
		return fmt.Errorf("rules.depth %d: must be >= 1", c.Rules.Depth)
	}
	if c.Policy.MaxBytes < 0 || c.Policy.MaxRepeat < 0 {
		return fmt.Errorf("policy.max_bytes and policy.max_repeat must be nonnegative")
	}
	if c.Policy.MinLen < 0 || c.Policy.MaxLen < 0 {
		return fmt.Errorf("policy lengths must be >= 0")
	}
	if c.Policy.MaxLen > 0 && c.Policy.MinLen > c.Policy.MaxLen {
		return fmt.Errorf("policy.min_len %d > policy.max_len %d", c.Policy.MinLen, c.Policy.MaxLen)
	}
	for _, r := range c.Policy.Require {
		switch r {
		case "upper", "lower", "digit", "special":
		default:
			return fmt.Errorf("policy.require %q: must be upper|lower|digit|special", r)
		}
	}
	return nil
}

func (p Profile) isEmpty() bool {
	return p.FirstName == "" && p.LastName == "" && p.Nickname == "" &&
		p.DOB == "" && p.Partner == "" && p.Company == "" && p.Domain == "" &&
		len(p.Pets) == 0 && len(p.Keywords) == 0
}

// ParsePolicy loads a standalone policy document with the same strict validation as a config.
func ParsePolicy(raw []byte) (Policy, error) {
	var p Policy
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("parse policy: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return p, fmt.Errorf("policy must contain exactly one YAML document")
	}
	cfg := Config{Profile: Profile{Keywords: []string{"validation"}}, Policy: p}
	cfg.applyDefaults()
	return p, cfg.validate()
}
