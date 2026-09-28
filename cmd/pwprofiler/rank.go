package main

import (
	"fmt"
	"io"
	"os"

	"github.com/praneeth132006/Password-Profiler/internal/rank"
	"github.com/spf13/cobra"
)

// newRankCmd reorders an existing wordlist so the most likely passwords come
// first. It is decoupled from generation: it ranks any newline-delimited list
// (this tool's output or one you already have) and, with --top, keeps only the
// K best using O(K) memory — so it works no matter how long the input is.
func newRankCmd() *cobra.Command {
	var input, output string
	var top int
	cmd := &cobra.Command{
		Use:   "rank",
		Short: "Reorder a wordlist so the most likely passwords come first",
		Long: "Rank a newline-delimited wordlist by estimated real-world likelihood,\n" +
			"most probable first. With --top N, only the N best are kept, using memory\n" +
			"proportional to N rather than the whole input — so it scales to lists of\n" +
			"any length. Ranking is an ordering signal, not a password-strength meter.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if top < 0 {
				return fmt.Errorf("--top must be >= 0 (0 ranks the entire list)")
			}

			var reader io.Reader = cmd.InOrStdin()
			if input != "-" {
				f, err := os.Open(input)
				if err != nil {
					return err
				}
				defer f.Close()
				reader = f
			}

			var writer io.Writer = cmd.OutOrStdout()
			if output != "" {
				f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
				if err != nil {
					return fmt.Errorf("create output %q (refusing to overwrite): %w", output, err)
				}
				defer f.Close()
				writer = f
			}

			n, err := rank.Stream(reader, writer, top)
			if err != nil {
				if output != "" {
					_ = os.Remove(output) // don't leave a partial ranked file
				}
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "pwprofiler: ranked %d candidates best-first\n", n)
			return nil
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", "-", "wordlist path, or - for stdin")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write ranked list to a new file (default stdout; refuses to overwrite)")
	cmd.Flags().IntVar(&top, "top", 0, "keep only the N most likely candidates (0 = rank the whole list)")
	return cmd
}
