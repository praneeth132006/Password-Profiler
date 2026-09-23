package main

import (
	"context"
	"sync"

	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
)

// tokenSource is a pull-style producer. A false callback stops upstream work.
type tokenSource func(context.Context, func(string) bool) error

type searchStats struct{ Filtered, Attempts, Unique, LimitedTokens, Oversized int }

func (s *searchStats) add(w mutate.WalkStats) {
	s.Attempts += w.Attempts
	s.Unique += w.Unique
	s.Oversized += w.Oversized
	if w.Limited {
		s.LimitedTokens++
	}
}

type generationReport struct{ SearchLimited bool }

func sliceSource(toks []string) tokenSource {
	return func(ctx context.Context, yield func(string) bool) error {
		for _, tok := range toks {
			if ctx.Err() != nil || !yield(tok) {
				break
			}
		}
		return ctx.Err()
	}
}

func expandSequential(toks []string, eng *mutate.Engine, pol *policy.Filter, w *output.Writer) (int, error) {
	return expandSequentialContext(context.Background(), toks, eng, pol, w)
}
func expandSequentialContext(ctx context.Context, toks []string, eng *mutate.Engine, pol *policy.Filter, w *output.Writer) (int, error) {
	stats, err := expandStream(ctx, sliceSource(toks), eng, pol, w, 1)
	return stats.Filtered, err
}

func expandStream(parent context.Context, source tokenSource, eng *mutate.Engine, pol *policy.Filter, w *output.Writer, workers int) (searchStats, error) {
	stats := searchStats{}
	if workers <= 1 {
		var generationErr error
		err := source(parent, func(tok string) bool {
			work, e := eng.Walk(parent, tok, func(c string) bool {
				if !pol.Allow(c) {
					stats.Filtered++
					return true
				}
				_, generationErr = w.Add(c)
				return generationErr == nil && !w.BudgetReached()
			})
			stats.add(work)
			if generationErr == nil {
				generationErr = e
			}
			return generationErr == nil && !w.BudgetReached()
		})
		if generationErr != nil {
			return stats, generationErr
		}
		return stats, err
	}
	// Workers stream fixed-size batches rather than materializing entire trees.
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	jobs := make(chan string)
	type message struct {
		values   []string
		work     mutate.WalkStats
		filtered int
	}
	results := make(chan message, workers)
	sourceErr := make(chan error, 1)
	go func() {
		defer close(jobs)
		sourceErr <- source(ctx, func(t string) bool {
			select {
			case jobs <- t:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for tok := range jobs {
				if ctx.Err() != nil {
					return
				}
				batch := make([]string, 0, 64)
				filtered := 0
				send := func() bool {
					if len(batch) == 0 {
						return true
					}
					select {
					case results <- message{values: batch}:
						batch = make([]string, 0, 64)
						return true
					case <-ctx.Done():
						return false
					}
				}
				work, _ := eng.Walk(ctx, tok, func(c string) bool {
					if !pol.Allow(c) {
						filtered++
						return true
					}
					batch = append(batch, c)
					if len(batch) == 64 {
						return send()
					}
					return true
				})
				if ctx.Err() == nil {
					send()
				}
				// The consumer drains stats even after cancellation; this send cannot leak.
				results <- message{work: work, filtered: filtered}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	var writeErr error
	for msg := range results {
		stats.add(msg.work)
		stats.Filtered += msg.filtered
		if writeErr != nil || w.BudgetReached() || parent.Err() != nil {
			cancel()
			continue
		}
		for _, c := range msg.values {
			_, writeErr = w.Add(c)
			if writeErr != nil || w.BudgetReached() {
				cancel()
				break
			}
		}
	}
	err := <-sourceErr
	if writeErr != nil {
		return stats, writeErr
	}
	if parent.Err() != nil {
		return stats, parent.Err()
	}
	if err != nil && err != context.Canceled {
		return stats, err
	}
	return stats, nil
}
