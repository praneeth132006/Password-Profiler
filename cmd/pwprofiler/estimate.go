package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/praneeth132006/Password-Profiler/internal/combine"
	"github.com/praneeth132006/Password-Profiler/internal/dedup"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"github.com/praneeth132006/Password-Profiler/internal/tokens"
	"github.com/spf13/cobra"
)

// newEstimateCmd answers the auditor's real question before a run: "how big is
// this, and is it worth it?" It counts the unique candidates a config produces
// (after policy) up to a cap, then reports the expected time to exhaust them at
// a given cracking rate.
func newEstimateCmd() *cobra.Command {
	var configPath string
	var hashrate float64
	var maxCount, workers int
	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate a config's unique candidate count and expected cracking effort",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if hashrate <= 0 {
				return fmt.Errorf("--hashrate must be > 0 (guesses per second)")
			}
			if maxCount < 1 {
				return fmt.Errorf("--max must be >= 1")
			}
			if workers < 1 || workers > 64 {
				return fmt.Errorf("--workers must be between 1 and 64")
			}
			cfg, err := profile.Load(configPath)
			if err != nil {
				return err
			}
			// Always bound the estimate so it terminates, regardless of config.
			cfg.Rules.Exhaustive = false

			base := tokens.Extract(cfg.Profile)
			if len(base) == 0 {
				return fmt.Errorf("inputs contain no usable tokens")
			}

			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			ctx, stop := signal.NotifyContext(parent, os.Interrupt)
			defer stop()

			var combined combine.Result
			source := func(sctx context.Context, yield func(string) bool) error {
				var e error
				combined, e = combine.Walk(sctx, base, combine.Config{
					RepeatTokens: cfg.Rules.RepeatTokens, MaxCombine: cfg.Rules.MaxCombine,
					Separators: cfg.Rules.Separators, Limit: cfg.Rules.CombineLimit,
					MaxAttempts: cfg.Rules.CombineAttempts, MaxBytes: cfg.Rules.MaxCandidateBytes,
				}, yield)
				return e
			}
			appendAffixes, prependAffixes := buildAffixes(cfg)
			eng := mutate.NewEngine(mutate.Config{
				LeetCap: cfg.Rules.LeetCap, MaxVariants: cfg.Rules.MaxVariants,
				MaxAttempts: cfg.Rules.MaxAttempts, MaxBytes: cfg.Rules.MaxCandidateBytes,
				Cases: cfg.Rules.Case, Leet: cfg.Rules.Leet,
				Append: appendAffixes, Prepend: prependAffixes,
				Structural: true, Depth: cfg.Rules.Depth,
			})
			pol := policy.FromProfile(cfg.Policy)

			// Count unique, policy-passing candidates into a discard sink.
			w := output.New(io.Discard, dedup.NewExact(), maxCount)
			search, err := expandStream(ctx, source, eng, pol, w, workers)
			if err != nil {
				return err
			}
			stats, err := w.Flush()
			if err != nil {
				return err
			}

			capped := stats.BudgetHit
			limited := search.LimitedTokens > 0 || combined.Capped || combined.WorkLimited
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Estimate for %s\n", configPath)
			fmt.Fprintf(out, "  base tokens:        %d\n", len(base))
			fmt.Fprintf(out, "  %s\n", combined.Describe())
			fmt.Fprintf(out, "  rejected by policy: %d\n", search.Filtered)
			countLabel := fmt.Sprintf("%d", stats.Emitted)
			if capped {
				countLabel = fmt.Sprintf("≥ %d (capped at --max; raise it for the full count)", stats.Emitted)
			} else if limited {
				countLabel = fmt.Sprintf("≥ %d (search limits reached; raise rules limits for the full count)", stats.Emitted)
			}
			fmt.Fprintf(out, "  unique candidates:  %s\n", countLabel)
			fmt.Fprintf(out, "  at %s guesses/sec:\n", humanRate(hashrate))
			exhaust := float64(stats.Emitted) / hashrate
			fmt.Fprintf(out, "    time to try all:  %s%s\n", humanDuration(exhaust), moreNote(capped || limited))
			fmt.Fprintf(out, "    expected 1st hit: ~%s (if the password is in the list)\n", humanDuration(exhaust/2))
			return nil
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to YAML config (required)")
	cmd.Flags().Float64Var(&hashrate, "hashrate", 1e9, "cracking rate in guesses/sec (e.g. 1e9 = 1 GH/s)")
	cmd.Flags().IntVar(&maxCount, "max", 20_000_000, "stop counting after this many unique candidates")
	cmd.Flags().IntVar(&workers, "workers", 1, "mutation worker goroutines used while counting")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

func moreNote(more bool) string {
	if more {
		return " (lower bound)"
	}
	return ""
}

// humanRate renders a guesses/sec value like "1.0 GH", "500.0 MH", "1.2 kH".
func humanRate(r float64) string {
	switch {
	case r >= 1e12:
		return fmt.Sprintf("%.1f TH", r/1e12)
	case r >= 1e9:
		return fmt.Sprintf("%.1f GH", r/1e9)
	case r >= 1e6:
		return fmt.Sprintf("%.1f MH", r/1e6)
	case r >= 1e3:
		return fmt.Sprintf("%.1f kH", r/1e3)
	default:
		return fmt.Sprintf("%.0f H", r)
	}
}

// humanDuration renders seconds as the largest natural unit.
func humanDuration(sec float64) string {
	switch {
	case sec < 1e-3:
		return "instant"
	case sec < 1:
		return fmt.Sprintf("%.0f ms", sec*1000)
	case sec < 60:
		return fmt.Sprintf("%.1f s", sec)
	case sec < 3600:
		return fmt.Sprintf("%.1f min", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%.1f h", sec/3600)
	case sec < 31557600:
		return fmt.Sprintf("%.1f days", sec/86400)
	default:
		return fmt.Sprintf("%.1f years", sec/31557600)
	}
}
