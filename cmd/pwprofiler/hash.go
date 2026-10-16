package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/praneeth132006/Password-Profiler/internal/hashes"
	"github.com/spf13/cobra"
)

// newHashCmd reads a newline-delimited wordlist and emits each candidate with
// its hash(es), for matching against hashes captured in an authorized
// engagement. It composes with the rest of the pipeline, e.g.:
//
//	pwprofiler generate -c audit.yaml | pwprofiler rank --top 1000 | pwprofiler hash --algos ntlm,md5
//
// Only fast, unsalted algorithms are supported; salted/slow schemes (bcrypt,
// argon2, ...) cannot be matched from a precomputed list and must be cracked
// directly with hashcat/John and a wordlist.
func newHashCmd() *cobra.Command {
	var input, output, sep string
	var algos []string
	cmd := &cobra.Command{
		Use:   "hash",
		Short: "Emit candidate:hash lines for a wordlist (fast/unsalted algorithms)",
		Long: "Hash every line of a wordlist with one or more fast, unsalted algorithms\n" +
			"(md5, sha1, sha256, sha512, ntlm) and print `candidate<sep>hash...` per line.\n" +
			"Use it to match a captured hash against your candidate list in an authorized\n" +
			"audit. Salted/slow hashes (bcrypt, argon2) are unsupported by design.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			set, err := hashes.NewSet(algos)
			if err != nil {
				return err
			}
			if set.Empty() {
				return fmt.Errorf("--algos is required (e.g. --algos md5,ntlm; supported: %s)", strings.Join(hashes.Supported(), ", "))
			}
			if sep == "" {
				sep = ":"
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

			bw := bufio.NewWriterSize(writer, 1<<16)
			sc := bufio.NewScanner(reader)
			sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
			n := 0
			for sc.Scan() {
				line := sc.Text()
				if line == "" {
					continue
				}
				if _, err := bw.WriteString(set.Line(line, sep)); err != nil {
					return err
				}
				if err := bw.WriteByte('\n'); err != nil {
					return err
				}
				n++
			}
			if err := sc.Err(); err != nil {
				if output != "" {
					_ = os.Remove(output)
				}
				return err
			}
			if err := bw.Flush(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "pwprofiler: hashed %d candidates (%s)\n", n, strings.Join(set.Names(), ", "))
			return nil
		},
	}
	cmd.Flags().StringVarP(&input, "input", "i", "-", "wordlist path, or - for stdin")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write output to a new file (default stdout; refuses to overwrite)")
	cmd.Flags().StringSliceVar(&algos, "algos", nil, "hash algorithms: md5,sha1,sha256,sha512,ntlm (comma-separated)")
	cmd.Flags().StringVar(&sep, "sep", ":", "field separator between candidate and its hashes")
	return cmd
}
