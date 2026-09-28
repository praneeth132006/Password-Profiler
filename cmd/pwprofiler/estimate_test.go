package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const estimateConfig = `profile:
  first_name: John
  last_name: Doe
  keywords: [falcons, chess]
rules:
  case: [lower, capitalize]
  leet: partial
  affixes:
    suffixes: ["1", "123", "!"]
policy:
  min_len: 8
  max_len: 16
  require: [upper, lower, digit]
`

func writeTempConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(estimateConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEstimateCommand(t *testing.T) {
	cfg := writeTempConfig(t)
	root := newRootCmd()
	root.SetArgs([]string{"estimate", "-c", cfg, "--hashrate", "1e4", "--max", "20000"})
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	if err := root.Execute(); err != nil {
		t.Fatalf("estimate failed: %v (stderr: %s)", err, errb.String())
	}
	s := out.String()
	for _, want := range []string{"base tokens:", "unique candidates:", "guesses/sec", "time to try all:", "expected 1st hit:"} {
		if !strings.Contains(s, want) {
			t.Errorf("estimate output missing %q\n%s", want, s)
		}
	}
}

func TestEstimateRejectsBadFlags(t *testing.T) {
	cfg := writeTempConfig(t)
	for _, args := range [][]string{
		{"estimate", "-c", cfg, "--hashrate", "0"},
		{"estimate", "-c", cfg, "--max", "0"},
	} {
		root := newRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}
}

func TestHumanDuration(t *testing.T) {
	cases := map[float64]string{
		0.0001:   "instant",
		0.5:      "500 ms",
		30:       "30.0 s",
		120:      "2.0 min",
		7200:     "2.0 h",
		172800:   "2.0 days",
		63115200: "2.0 years",
	}
	for sec, want := range cases {
		if got := humanDuration(sec); got != want {
			t.Errorf("humanDuration(%v) = %q, want %q", sec, got, want)
		}
	}
}

func TestHumanRate(t *testing.T) {
	cases := map[float64]string{
		1e12: "1.0 TH",
		1e9:  "1.0 GH",
		5e6:  "5.0 MH",
		2e3:  "2.0 kH",
		50:   "50 H",
	}
	for r, want := range cases {
		if got := humanRate(r); got != want {
			t.Errorf("humanRate(%v) = %q, want %q", r, got, want)
		}
	}
}
