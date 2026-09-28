package rank

import (
	"bufio"
	"container/heap"
	"fmt"
	"io"
	"sort"
)

// minHeap keeps the K highest-scoring candidates seen so far. Its top (index 0)
// is the WEAKEST retained candidate, so it is the first evicted once the heap is
// full. Ties keep the earliest-seen candidate (larger idx is evicted first).
type minHeap []scored

func (h minHeap) Len() int      { return len(h) }
func (h minHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h minHeap) Less(i, j int) bool {
	if h[i].score != h[j].score {
		return h[i].score < h[j].score
	}
	return h[i].idx > h[j].idx
}
func (h *minHeap) Push(x any) { *h = append(*h, x.(scored)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// Options tunes a ranking pass.
type Options struct {
	// Top keeps only the N highest-scoring candidates (0 = rank everything).
	Top int
	// Model, when non-nil, adds a frequency bonus on top of the heuristic Score.
	Model *Model
	// Exclude drops any candidate present in this set before ranking (e.g. a
	// list of already-tried passwords), so repeat audits only rank net-new ones.
	Exclude map[string]struct{}
}

// Stream ranks candidates from r to w best-first using the default heuristic.
// See StreamWith for model/exclusion options. Blank lines are skipped and
// duplicates are preserved (de-duplication is the generator's job).
func Stream(r io.Reader, w io.Writer, top int) (int, error) {
	return StreamWith(r, w, Options{Top: top})
}

// StreamWith ranks candidates from r and writes them to w ordered best-first.
// When Top > 0 it retains only the top-K using an O(K)-memory min-heap, so it
// ranks arbitrarily long inputs without buffering them all; when Top <= 0 it
// buffers and ranks every line. Returns the number of lines written.
func StreamWith(r io.Reader, w io.Writer, opts Options) (int, error) {
	top := opts.Top
	scoreOf := Score
	if opts.Model != nil {
		scoreOf = opts.Model.Score
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		h   minHeap
		all []scored
		idx int
	)
	if top > 0 {
		h = make(minHeap, 0, top+1)
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if opts.Exclude != nil {
			if _, skip := opts.Exclude[line]; skip {
				continue
			}
		}
		item := scored{line: line, score: scoreOf(line), idx: idx}
		idx++
		if top <= 0 {
			all = append(all, item)
			continue
		}
		if len(h) < top {
			heap.Push(&h, item)
			continue
		}
		// Replace the weakest retained candidate if this one is stronger. On an
		// exact tie the earlier candidate (already in the heap) wins.
		if item.score > h[0].score {
			h[0] = item
			heap.Fix(&h, 0)
		}
	}
	if err := sc.Err(); err != nil {
		return 0, fmt.Errorf("read candidates: %w", err)
	}

	items := all
	if top > 0 {
		items = []scored(h)
	}
	sort.SliceStable(items, func(i, j int) bool { return less(items[i], items[j]) })

	bw := bufio.NewWriter(w)
	written := 0
	for _, it := range items {
		if _, err := bw.WriteString(it.line); err != nil {
			return written, fmt.Errorf("write candidate: %w", err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			return written, fmt.Errorf("write newline: %w", err)
		}
		written++
	}
	if err := bw.Flush(); err != nil {
		return written, fmt.Errorf("flush: %w", err)
	}
	return written, nil
}
