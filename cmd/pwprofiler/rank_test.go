package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRankCommandStdinStdout(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"rank", "--top", "2"})
	root.SetIn(strings.NewReader("x\nFalcons2024!\naaaaaaaa\nNorthstar1\n"))
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	if err := root.Execute(); err != nil {
		t.Fatalf("rank failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 ranked lines, got %d: %v", len(lines), lines)
	}
	// The two strongest candidates should be selected, best first.
	if lines[0] != "Falcons2024!" && lines[0] != "Northstar1" {
		t.Errorf("unexpected top candidate %q", lines[0])
	}
	for _, weak := range []string{"x", "aaaaaaaa"} {
		if strings.Contains(out.String(), weak) {
			t.Errorf("weak candidate %q should not be in the top 2", weak)
		}
	}
}

func TestRankCommandRejectsNegativeTop(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"rank", "--top", "-1"})
	root.SetIn(strings.NewReader("a\n"))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for negative --top")
	}
}
