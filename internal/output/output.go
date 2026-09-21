// Package output is the sink for candidate passwords. It wraps a bufio.Writer
// (all candidate writes are buffered — never raw), optionally de-duplicates,
// enforces the global budget cap, and tracks stats for a summary line.
package output

import (
	"bufio"
	"fmt"
	"io"
)

// Stats summarizes a run for the final report.
type Stats struct {
	Emitted    int  // lines actually written
	Duplicates int  // candidates suppressed by dedup
	BudgetHit  bool // true if the global budget stopped emission early
	budget     int
}

// Writer buffers, dedupes, and budget-caps candidate output.
type Writer struct {
	bw     *bufio.Writer
	dedupe bool
	seen   map[string]struct{}
	stats  Stats
}

// New builds a Writer over w. If dedupe is true, exact duplicates are dropped
// (Phase 1 uses an in-memory set; Phase 6 replaces this with a scalable dedup).
// budget <= 0 means unlimited.
func New(w io.Writer, dedupe bool, budget int) *Writer {
	var seen map[string]struct{}
	if dedupe {
		seen = make(map[string]struct{})
	}
	return &Writer{
		bw:     bufio.NewWriterSize(w, 1<<16),
		dedupe: dedupe,
		seen:   seen,
		stats:  Stats{budget: budget},
	}
}

// Add writes a single candidate. It returns false once the budget is reached,
// signalling the caller to stop feeding candidates. Errors are I/O errors.
func (w *Writer) Add(candidate string) (accepted bool, err error) {
	if w.stats.budget > 0 && w.stats.Emitted >= w.stats.budget {
		w.stats.BudgetHit = true
		return false, nil
	}
	if w.dedupe {
		if _, ok := w.seen[candidate]; ok {
			w.stats.Duplicates++
			return true, nil // accepted (not a stop condition), just suppressed
		}
		w.seen[candidate] = struct{}{}
	}
	if _, err := w.bw.WriteString(candidate); err != nil {
		return false, fmt.Errorf("write candidate: %w", err)
	}
	if err := w.bw.WriteByte('\n'); err != nil {
		return false, fmt.Errorf("write newline: %w", err)
	}
	w.stats.Emitted++
	if w.stats.budget > 0 && w.stats.Emitted >= w.stats.budget {
		w.stats.BudgetHit = true
	}
	return true, nil
}

// Flush flushes the underlying buffer and returns the final stats.
func (w *Writer) Flush() (Stats, error) {
	if err := w.bw.Flush(); err != nil {
		return w.stats, fmt.Errorf("flush output: %w", err)
	}
	return w.stats, nil
}

// BudgetReached reports whether the budget cap has been hit.
func (w *Writer) BudgetReached() bool { return w.stats.BudgetHit }
