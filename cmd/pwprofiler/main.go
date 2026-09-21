// Command pwprofiler generates targeted password-audit wordlists (and, from
// Phase 4, hashcat rule files) from OSINT about an authorized target.
//
// AUTHORIZED TESTING ONLY. This tool is for password auditing and penetration
// testing that you are explicitly authorized to perform. Misuse against systems
// or accounts you do not own or have written permission to test may be illegal.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/praneeth132006/Password-Profiler/internal/combine"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"github.com/praneeth132006/Password-Profiler/internal/rules"
	"github.com/praneeth132006/Password-Profiler/internal/tokens"
	"github.com/spf13/cobra"
)

// version is overrideable at build time via -ldflags "-X main.version=...".
var version = "0.1.0-phase1"

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
	root.AddCommand(newGenerateCmd())
	return root
}

func newGenerateCmd() *cobra.Command {
	var (
		configPath string
		outPath    string
		modeFlag   string
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
			return runGenerate(cmd, cfg)
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to YAML config (required)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "write candidates here instead of the config's output.file / stdout")
	cmd.Flags().StringVar(&modeFlag, "mode", "", "override output mode: wordlist|rules")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}

// runGenerate executes the Phase 1 pipeline: tokens -> mutate -> buffered output.
func runGenerate(cmd *cobra.Command, cfg *profile.Config) error {
	switch cfg.Output.Mode {
	case "wordlist":
		return runWordlist(cmd, cfg)
	case "rules":
		return runRules(cmd, cfg)
	default:
		return fmt.Errorf("output.mode %q: must be wordlist|rules", cfg.Output.Mode)
	}
}

// runWordlist runs the full pipeline and writes every candidate.
func runWordlist(cmd *cobra.Command, cfg *profile.Config) error {
	// Resolve the destination: a file, or stdout. Candidates go to the sink;
	// the stats summary always goes to stderr so a stdout pipe stays clean.
	var sink *os.File
	if cfg.Output.File != "" {
		f, err := os.Create(cfg.Output.File)
		if err != nil {
			return fmt.Errorf("create output file %q: %w", cfg.Output.File, err)
		}
		defer f.Close()
		sink = f
	} else {
		sink = os.Stdout
	}

	base := tokens.Extract(cfg.Profile)
	combined := combine.Generate(base, combine.Config{
		MaxCombine: cfg.Rules.MaxCombine,
		Separators: cfg.Rules.Separators,
		Limit:      cfg.Output.Budget, // guard against combination explosion
	})
	appendAffixes, prependAffixes := buildAffixes(cfg)
	eng := mutate.NewEngine(mutate.Config{
		Cases:      cfg.Rules.Case,
		Leet:       cfg.Rules.Leet,
		Append:     appendAffixes,
		Prepend:    prependAffixes,
		Structural: true,
		Depth:      cfg.Rules.Depth,
	})
	pol := policy.New(policy.Policy{
		MinLen:  cfg.Policy.MinLen,
		MaxLen:  cfg.Policy.MaxLen,
		Require: cfg.Policy.Require,
	})
	w := output.New(sink, cfg.Output.Dedupe, cfg.Output.Budget)

	var filtered int
	for _, tok := range combined.Tokens {
		for _, cand := range eng.Expand(tok) {
			if pol.Active() && !pol.Allow(cand) {
				filtered++
				continue
			}
			accepted, err := w.Add(cand)
			if err != nil {
				return err
			}
			if !accepted { // budget reached
				break
			}
		}
		if w.BudgetReached() {
			break
		}
	}

	stats, err := w.Flush()
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.ErrOrStderr(),
		"pwprofiler: %d base tokens -> %s -> %d candidates emitted (%d duplicates suppressed%s)%s\n",
		len(base), combined.Describe(), stats.Emitted, stats.Duplicates,
		policyNote(pol, filtered), budgetNote(stats.BudgetHit, cfg.Output.Budget))
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
func runRules(cmd *cobra.Command, cfg *profile.Config) error {
	wordPath := cfg.Output.File
	if wordPath == "" {
		wordPath = "pwprofiler.words"
	}
	rulePath := deriveRulePath(wordPath)

	// Base wordlist: the combined tokens, buffered + deduped, budget-capped.
	base := tokens.Extract(cfg.Profile)
	combined := combine.Generate(base, combine.Config{
		MaxCombine: cfg.Rules.MaxCombine,
		Separators: cfg.Rules.Separators,
		Limit:      cfg.Output.Budget,
	})
	wf, err := os.Create(wordPath)
	if err != nil {
		return fmt.Errorf("create wordlist %q: %w", wordPath, err)
	}
	defer wf.Close()
	ww := output.New(wf, cfg.Output.Dedupe, cfg.Output.Budget)
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
	rf, err := os.Create(rulePath)
	if err != nil {
		return fmt.Errorf("create rule file %q: %w", rulePath, err)
	}
	defer rf.Close()
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

func budgetNote(hit bool, budget int) string {
	if hit {
		return fmt.Sprintf(" [BUDGET CAP of %d reached — output truncated]", budget)
	}
	return ""
}
