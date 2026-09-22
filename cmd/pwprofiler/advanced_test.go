package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func TestStreamingCheckPreservesPasswordCharacters(t *testing.T) {
	p := policy.New(policy.Policy{MinLen: 3, Blocklist: []string{" # ", "abc"}})
	report, err := checkWords(strings.NewReader(" # \r\nabc\n abc \n\nééé\n"), p)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 5 || report.Accepted != 2 || report.Rejected != 3 || report.Reasons["blocklisted"] != 2 || report.Reasons["min_length"] != 1 {
		t.Fatalf("unexpected report %+v", report)
	}
}
func TestCheckCommandReportAndExit(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "policy.yaml")
	report := filepath.Join(dir, "report.json")
	if err := os.WriteFile(p, []byte("min_len: 8\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	cmd.SetArgs([]string{"check", "--policy", p, "--report", report, "--fail-on-reject"})
	cmd.SetIn(strings.NewReader("short\nNorthstar\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected rejection exit")
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"rejected": 1`)) || bytes.Contains(data, []byte("Northstar")) {
		t.Fatal(string(data))
	}
}
func TestSessionPersistence(t *testing.T) {
	s := newSession()
	s.Keywords = []string{"Northstar", "Riverstone"}
	s.Sources = []string{"manual"}
	s.Blocklist = []string{"Northstar1!"}
	s.MaxBytes = 72
	s.MaxRepeat = 2
	s.Forbidden = "<>"
	path := filepath.Join(t.TempDir(), "session.json")
	if err := s.save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSession(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, loaded) {
		t.Fatalf("roundtrip mismatch: %+v", loaded)
	}
	if err = s.save(path); err == nil {
		t.Fatal("overwrote saved session")
	}
	for _, raw := range []string{`{"schema":2,"session":{}}`, `{"schema":1,"unknown":true}`, `{} {}`} {
		p := filepath.Join(t.TempDir(), "bad.json")
		_ = os.WriteFile(p, []byte(raw), 0600)
		if _, err = loadSession(p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
func TestStandalonePolicyValidation(t *testing.T) {
	for _, raw := range []string{"max_bytes: -1", "max_repeat: -1", "require: [typo]", "min_len: 20\nmax_len: 10", "typo: true", "min_len: 8\n---\nmax_len: 20"} {
		if _, err := profile.ParsePolicy([]byte(raw)); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}
func TestPolicyFileGeneration(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	output := filepath.Join(dir, "out.txt")
	path := filepath.Join(dir, "policy.yaml")
	_ = os.WriteFile(input, []byte("Northstar"), 0600)
	_ = os.WriteFile(path, []byte("min_len: 8\nmax_len: 64\nforbidden: '!'\nblocklist: [northstar]\n"), 0600)
	cmd := newRootCmd()
	cmd.SetArgs([]string{"files", "-i", input, "--policy", path, "-o", output})
	var log bytes.Buffer
	cmd.SetErr(&log)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || bytes.Contains(data, []byte("!")) {
		t.Fatalf("policy not enforced %s", data)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "northstar" {
			t.Fatal("blocklisted word emitted")
		}
	}
}

func TestEmptyWordlistAndOversizedLinesFail(t *testing.T) {
	for _, text := range []string{"", strings.Repeat("x", 1<<20)} {
		if _, err := checkWords(strings.NewReader(text), policy.New(policy.Policy{})); err == nil {
			t.Fatal("invalid wordlist accepted")
		}
	}
}

func TestValidateHasNoOutputSideEffects(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "passwords.txt")
	config := filepath.Join(dir, "config.yaml")
	contents := "profile:\n  keywords: [Northstar]\noutput:\n  file: " + dest + "\n"
	if err := os.WriteFile(config, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newRootCmd()
	var result bytes.Buffer
	cmd.SetOut(&result)
	cmd.SetArgs([]string{"validate", "-c", config})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("validation created output")
	}
	if !strings.Contains(result.String(), "Valid config") {
		t.Fatal(result.String())
	}
}
