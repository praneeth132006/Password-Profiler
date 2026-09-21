package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriterBasic(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, false, 0)
	for _, c := range []string{"a", "b", "c"} {
		if ok, err := w.Add(c); err != nil || !ok {
			t.Fatalf("Add(%q) = %v, %v", c, ok, err)
		}
	}
	stats, err := w.Flush()
	if err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got := buf.String(); got != "a\nb\nc\n" {
		t.Errorf("output = %q, want a\\nb\\nc\\n", got)
	}
	if stats.Emitted != 3 {
		t.Errorf("emitted = %d, want 3", stats.Emitted)
	}
}

func TestWriterDedupe(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, true, 0)
	for _, c := range []string{"x", "x", "y", "x"} {
		if _, err := w.Add(c); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	stats, _ := w.Flush()
	if got := buf.String(); got != "x\ny\n" {
		t.Errorf("output = %q, want x\\ny\\n", got)
	}
	if stats.Emitted != 2 {
		t.Errorf("emitted = %d, want 2", stats.Emitted)
	}
	if stats.Duplicates != 2 {
		t.Errorf("duplicates = %d, want 2", stats.Duplicates)
	}
}

func TestWriterBudget(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, false, 2)
	accepted := 0
	for _, c := range []string{"a", "b", "c", "d"} {
		ok, err := w.Add(c)
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if ok {
			accepted++
		} else {
			break
		}
	}
	stats, _ := w.Flush()
	if stats.Emitted != 2 {
		t.Errorf("emitted = %d, want 2 (budget)", stats.Emitted)
	}
	if !stats.BudgetHit || !w.BudgetReached() {
		t.Errorf("expected BudgetHit true")
	}
	if lines := strings.Count(buf.String(), "\n"); lines != 2 {
		t.Errorf("wrote %d lines, want 2", lines)
	}
}
