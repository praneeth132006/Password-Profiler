// Package output is the sink for candidate passwords. It wraps a bufio.Writer
// (all candidate writes are buffered — never raw), optionally de-duplicates,
// enforces the global budget cap, and tracks stats for a summary line.
package output

import (
	"bufio"
	"fmt"
	"io"

	"github.com/praneeth132006/Password-Profiler/internal/dedup"
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
	bw    *bufio.Writer
	dd    dedup.Deduper // nil => no de-duplication
	stats Stats
}

// New builds a Writer over w. Pass a dedup.Deduper to drop duplicates (an
// Exact set for correctness, or a Bloom filter for bounded memory at scale);
// pass nil to disable de-duplication. budget <= 0 means unlimited.
func New(w io.Writer, dd dedup.Deduper, budget int) *Writer {
	return &Writer{
		bw:    bufio.NewWriterSize(w, 1<<16),
		dd:    dd,
		stats: Stats{budget: budget},
	}
}

// DedupKind reports the de-dup strategy in use ("exact", "bloom", or "none").
func (w *Writer) DedupKind() string {
	if w.dd == nil {
		return "none"
	}
	return w.dd.Kind()
}

// Add writes a single candidate. It returns false once the budget is reached,
// signalling the caller to stop feeding candidates. Errors are I/O errors.
func (w *Writer) Add(candidate string) (accepted bool, err error) {
	if w.stats.budget > 0 && w.stats.Emitted >= w.stats.budget {
		w.stats.BudgetHit = true
		return false, nil
	}
	if w.dd != nil {
		if w.dd.Seen(candidate) {
			w.stats.Duplicates++
			return true, nil // accepted (not a stop condition), just suppressed
		}
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
