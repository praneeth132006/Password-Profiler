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

	"github.com/praneeth132006/Password-Profiler/internal/combine"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
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
	if cfg.Output.Mode == "rules" {
		return fmt.Errorf("output mode %q is not implemented yet (arrives in Phase 4); run with --mode wordlist", cfg.Output.Mode)
	}

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
	eng := mutate.NewEngine(mutate.Config{
		Cases:    cfg.Rules.Case,
		Suffixes: cfg.Rules.Affixes.Suffixes,
	})
	w := output.New(sink, cfg.Output.Dedupe, cfg.Output.Budget)

	for _, tok := range combined.Tokens {
		for _, cand := range eng.Expand(tok) {
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
		"pwprofiler: %d base tokens -> %s -> %d candidates emitted (%d duplicates suppressed)%s\n",
		len(base), combined.Describe(), stats.Emitted, stats.Duplicates,
		budgetNote(stats.BudgetHit, cfg.Output.Budget))
	return nil
}

func budgetNote(hit bool, budget int) string {
	if hit {
		return fmt.Sprintf(" [BUDGET CAP of %d reached — output truncated]", budget)
	}
	return ""
}
