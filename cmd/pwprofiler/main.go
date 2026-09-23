// Command pwprofiler generates targeted password-audit wordlists (and, from
// Phase 4, hashcat rule files) from OSINT about an authorized target.
//
// AUTHORIZED TESTING ONLY. This tool is for password auditing and penetration
// testing that you are explicitly authorized to perform. Misuse against systems
// or accounts you do not own or have written permission to test may be illegal.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/praneeth132006/Password-Profiler/internal/combine"
	"github.com/praneeth132006/Password-Profiler/internal/dedup"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"github.com/praneeth132006/Password-Profiler/internal/rules"
	"github.com/praneeth132006/Password-Profiler/internal/tokens"
	"github.com/spf13/cobra"
)

// version is overrideable at build time via -ldflags "-X main.version=...".
var version = "0.4.0"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// Cobra already prints the error; exit non-zero for scripts/pipes.
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pwprofiler",
		Short: "Targeted password-profiling wordlist generator (authorized testing only)",
		Long: "pwprofiler builds candidate password wordlists from OSINT about an\n" +
			"authorized target. AUTHORIZED TESTING ONLY.",
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       version,
	}
	root.RunE = runConsole
	root.AddCommand(newGenerateCmd(), newConsoleCmd(), newFilesCmd(), newServeCmd(), newCheckCmd(), newValidateCmd())
	return root
}

// genOpts carries CLI-only knobs (not part of the config schema) for scale.
type genOpts struct {
	workers int // mutation worker goroutines (>1 trades output ordering for speed)
	report  *generationReport
	dedup   string // auto | exact | bloom
}

func newGenerateCmd() *cobra.Command {
	var (
		configPath string
		outPath    string
		modeFlag   string
		opts       genOpts
	)
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate candidates from a YAML config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := profile.Load(configPath)
			if err != nil {
				return err
			}
			// Flag overrides.
			if modeFlag != "" {
				cfg.Output.Mode = modeFlag
			}
			if outPath != "" {
				cfg.Output.File = outPath
			}
			if opts.workers < 1 || opts.workers > 64 {
				return fmt.Errorf("--workers must be between 1 and 64")
			}
			switch opts.dedup {
			case "", "auto", "exact", "bloom":
			default:
				return fmt.Errorf("--dedup %q: must be auto|exact|bloom", opts.dedup)
			}
			return runGenerate(cmd, cfg, opts)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to YAML config (required)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "write candidates here instead of the config's output.file / stdout")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "override output mode: wordlist|rules")
	cmd.Flags().IntVar(&opts.workers, "workers", 1, "mutation worker goroutines (>1 speeds up large runs but does not preserve output order)")
	cmd.Flags().StringVar(&opts.dedup, "dedup", "auto", "dedup strategy: auto|exact|bloom (auto uses bloom for large budgets)")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

// runGenerate dispatches on the output mode.
func runGenerate(cmd *cobra.Command, cfg *profile.Config, opts genOpts) error {
	switch cfg.Output.Mode {
	case "wordlist":
		return runWordlist(cmd, cfg, opts)
	case "rules":
		if policy.FromProfile(cfg.Policy).Active() {
			return fmt.Errorf("password policy is supported only in wordlist mode; rules cannot enforce it")
		}
		return runRules(cmd, cfg, opts)
	default:
		return fmt.Errorf("output.mode %q: must be wordlist|rules", cfg.Output.Mode)
	}
}

// bloomAutoThreshold is the budget at/above which "auto" dedup picks the
// memory-bounded Bloom filter over an exact set.
const bloomAutoThreshold = 1_000_000

// buildDeduper constructs the de-dup strategy from config + the --dedup flag.
// Returns nil when de-duplication is disabled.
func buildDeduper(cfg *profile.Config, strategy string) dedup.Deduper {
	if !cfg.Output.Dedupe {
		return nil
	}
	budget := cfg.Output.Budget
	if budget <= 0 {
		budget = 5_000_000
	}
	switch strategy {
	case "exact":
		return dedup.NewExact()
	case "bloom":
		return dedup.NewBloom(budget, 0.001)
	default: // auto
		if budget >= bloomAutoThreshold {
			return dedup.NewBloom(budget, 0.001)
		}
		return dedup.NewExact()
	}
}

// runWordlist runs the full pipeline and writes every candidate.
func runWordlist(cmd *cobra.Command, cfg *profile.Config, opts genOpts) (retErr error) {
	base := tokens.Extract(cfg.Profile)
	if len(base) == 0 {
		return fmt.Errorf("inputs contain no usable tokens")
	}
	// Resolve the destination: a file, or stdout. Candidates go to the sink;
	// the stats summary always goes to stderr so a stdout pipe stays clean.
	var sink io.Writer
	if cfg.Output.File != "" {
		f, err := os.OpenFile(cfg.Output.File, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return fmt.Errorf("create output file %q: %w", cfg.Output.File, err)
		}
		defer func() {
			closeErr := f.Close()
			if retErr == nil {
				retErr = closeErr
			}
			if retErr != nil {
				_ = os.Remove(cfg.Output.File)
			}
		}()
		sink = f
	} else {
		sink = cmd.OutOrStdout()
	}

	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt)
	defer stop()
	var combined combine.Result
	source := func(sourceCtx context.Context, yield func(string) bool) error {
		var err error
		combined, err = combine.Walk(sourceCtx, base, combine.Config{MaxCombine: cfg.Rules.MaxCombine, Separators: cfg.Rules.Separators, Limit: cfg.Rules.CombineLimit, MaxAttempts: cfg.Rules.CombineAttempts, MaxBytes: cfg.Rules.MaxCandidateBytes}, yield)
		return err
	}

	appendAffixes, prependAffixes := buildAffixes(cfg)
	eng := mutate.NewEngine(mutate.Config{
		MaxVariants: cfg.Rules.MaxVariants, MaxAttempts: cfg.Rules.MaxAttempts, MaxBytes: cfg.Rules.MaxCandidateBytes,
		Cases:      cfg.Rules.Case,
		Leet:       cfg.Rules.Leet,
		Append:     appendAffixes,
		Prepend:    prependAffixes,
		Structural: true,
		Depth:      cfg.Rules.Depth,
	})
	pol := policy.FromProfile(cfg.Policy)
	w := output.New(sink, buildDeduper(cfg, opts.dedup), cfg.Output.Budget)

	start := time.Now()
	search, err := expandStream(ctx, source, eng, pol, w, opts.workers)
	limited := search.LimitedTokens > 0 || search.Oversized > 0 || combined.Capped || combined.WorkLimited || combined.Oversized > 0
	if opts.report != nil {
		opts.report.SearchLimited = limited
	}
	if limited {
		fmt.Fprintf(cmd.ErrOrStderr(), "pwprofiler: search limits reached (mutation-limited tokens=%d; oversized mutations=%d; %s); coverage is incomplete\n", search.LimitedTokens, search.Oversized, combined.Describe())
	}

	if err != nil {
		return err
	}
	stats, err := w.Flush()
	if err != nil {
		return err
	}
	if stats.Emitted == 0 {
		if limited {
			return fmt.Errorf("no matching candidates found before search limits were reached; increase search limits or adjust inputs")
		}
		return fmt.Errorf("no generated candidates match the password policy; add input words or adjust the policy")
	}
	elapsed := time.Since(start)

	fmt.Fprintf(cmd.ErrOrStderr(),
		"pwprofiler: %d base tokens -> %s -> %d candidates emitted (%d duplicates suppressed%s)%s\n",
		len(base), combined.Describe(), stats.Emitted, stats.Duplicates,
		policyNote(pol, search.Filtered), budgetNote(stats.BudgetHit, cfg.Output.Budget))
	fmt.Fprintf(cmd.ErrOrStderr(),
		"           dedup=%s workers=%d, %s in %s (%.0f cand/s)\n",
		w.DedupKind(), opts.workers, humanCount(stats.Emitted), elapsed.Round(time.Millisecond),
		ratePerSec(stats.Emitted, elapsed))
	return nil
}

// policyNote reports how many candidates the policy filter rejected, when active.
func policyNote(pol *policy.Filter, filtered int) string {
	if !pol.Active() {
		return ""
	}
	return fmt.Sprintf(", %d rejected by policy", filtered)
}

// runRules is the headline feature: instead of materializing a giant wordlist,
// it writes a small base wordlist (the combined tokens) plus a hashcat .rule
// file that encodes the mutation set. hashcat expands base × rules on the fly.
func runRules(cmd *cobra.Command, cfg *profile.Config, opts genOpts) (retErr error) {
	wordPath := cfg.Output.File
	if wordPath == "" {
		wordPath = "pwprofiler.words"
	}
	rulePath := deriveRulePath(wordPath)
	if rulePath == wordPath {
		return fmt.Errorf("wordlist output must not have a .rule extension")
	}

	// Base wordlist: the combined tokens, buffered + deduped, budget-capped.
	base := tokens.Extract(cfg.Profile)
	combined := combine.Generate(base, combine.Config{
		MaxCombine:  cfg.Rules.MaxCombine,
		Separators:  cfg.Rules.Separators,
		Limit:       cfg.Rules.CombineLimit,
		MaxAttempts: cfg.Rules.CombineAttempts, MaxBytes: cfg.Rules.MaxCandidateBytes,
	})
	wf, err := os.OpenFile(wordPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create wordlist %q: %w", wordPath, err)
	}
	defer func() {
		err := wf.Close()
		if retErr == nil {
			retErr = err
		}
		if retErr != nil {
			_ = os.Remove(wordPath)
		}
	}()
	ww := output.New(wf, buildDeduper(cfg, opts.dedup), cfg.Output.Budget)
	for _, tok := range combined.Tokens {
		if ok, err := ww.Add(tok); err != nil {
			return err
		} else if !ok {
			break
		}
	}
	wstats, err := ww.Flush()
	if err != nil {
		return err
	}

	// Rule file encoding the mutation set.
	appendAffixes, prependAffixes := buildAffixes(cfg)
	ruleSet := rules.Generate(rules.Config{
		Cases:      cfg.Rules.Case,
		Leet:       cfg.Rules.Leet,
		Append:     appendAffixes,
		Prepend:    prependAffixes,
		Structural: true,
	})
	rf, err := os.OpenFile(rulePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create rule file %q: %w", rulePath, err)
	}
	defer func() {
		err := rf.Close()
		if retErr == nil {
			retErr = err
		}
		if retErr != nil {
			_ = os.Remove(rulePath)
		}
	}()
	if _, err := ruleSet.WriteTo(rf); err != nil {
		return err
	}

	// Report artifacts, estimated keyspace, and the hashcat invocation.
	keyspace := int64(wstats.Emitted) * int64(ruleSet.Len())
	out := cmd.ErrOrStderr()
	fmt.Fprintf(out, "pwprofiler (rules mode):\n")
	fmt.Fprintf(out, "  wordlist: %s  (%d words)%s\n", wordPath, wstats.Emitted,
		budgetNote(wstats.BudgetHit, cfg.Output.Budget))
	fmt.Fprintf(out, "  rules:    %s  (%d rules)\n", rulePath, ruleSet.Len())
	fmt.Fprintf(out, "  estimated keyspace: ~%d candidates (words × rules)\n", keyspace)
	fmt.Fprintf(out, "  run: hashcat -a 0 -m <hash-type> <hashes> %s -r %s\n", wordPath, rulePath)
	return nil
}

// deriveRulePath turns a wordlist path into a sibling .rule path.
func deriveRulePath(wordPath string) string {
	if ext := filepath.Ext(wordPath); ext != "" {
		return strings.TrimSuffix(wordPath, ext) + ".rule"
	}
	return wordPath + ".rule"
}

// buildAffixes assembles the append and prepend affix lists from config. Years
// (4- and 2-digit) are derived from the DOB and/or explicit extras; numeric and
// symbol suffixes come straight from config. Years are also prepended (4-digit
// only) since year prefixes are common ("1990john").
func buildAffixes(cfg *profile.Config) (appendList, prependList []string) {
	y := cfg.Rules.Affixes.Years
	var years []string
	switch {
	case y.FromDOB:
		var birthYear int
		if t, err := time.Parse("2006-01-02", cfg.Profile.DOB); err == nil {
			birthYear = t.Year()
		}
		years = mutate.YearAffixes(birthYear, birthYear > 0, time.Now().Year(), y.Extra)
	case len(y.Extra) > 0:
		years = mutate.YearAffixes(0, false, 0, y.Extra)
	}

	appendList = append(appendList, years...)
	appendList = append(appendList, cfg.Rules.Affixes.Suffixes...)
	for _, yr := range years {
		if len(yr) == 4 {
			prependList = append(prependList, yr)
		}
	}
	return appendList, prependList
}

// ratePerSec computes candidates emitted per second (0 for a zero duration).
func ratePerSec(n int, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / d.Seconds()
}

// humanCount renders a count compactly (1234567 -> "1.2M").
func humanCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM candidates", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK candidates", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d candidates", n)
	}
}

func budgetNote(hit bool, budget int) string {
	if hit {
		return fmt.Sprintf(" [BUDGET CAP of %d reached — output truncated]", budget)
	}
	return ""
}
