package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"github.com/praneeth132006/Password-Profiler/internal/tokens"
	"github.com/spf13/cobra"
)

// checkReport intentionally contains no passwords, samples, or source paths.
type checkReport struct {
	Total    int            `json:"total"`
	Accepted int            `json:"accepted"`
	Rejected int            `json:"rejected"`
	Reasons  map[string]int `json:"first_failure_counts"`
}

func checkWords(r io.Reader, p *policy.Filter) (checkReport, error) {
	report := checkReport{Reasons: map[string]int{}}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		// ScanLines removes CRLF but preserves password whitespace and # characters.
		word := scanner.Text()
		report.Total++
		reason := p.Reason(word)
		if reason == "" {
			report.Accepted++
		} else {
			report.Rejected++
			report.Reasons[reason]++
		}
	}
	if err := scanner.Err(); err != nil {
		return report, fmt.Errorf("read wordlist (maximum line size 1 MiB): %w", err)
	}
	if report.Total == 0 {
		return report, fmt.Errorf("wordlist contains no candidates")
	}
	return report, nil
}
func loadPolicyFile(path string) (profile.Policy, error) {
	var p profile.Policy
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 4<<20+1))
	if err != nil {
		return p, err
	}
	if len(data) > 4<<20 {
		return p, fmt.Errorf("policy exceeds 4 MiB")
	}
	return profile.ParsePolicy(data)
}
func writePrivate(path string, data []byte) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(path)
		}
	}()
	_, err = f.Write(data)
	return err
}
func newCheckCmd() *cobra.Command {
	var input, policyPath, reportPath string
	var fail bool
	cmd := &cobra.Command{Use: "check", Short: "Stream an existing wordlist through a policy; report counts without exposing passwords", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		p, err := loadPolicyFile(policyPath)
		if err != nil {
			return err
		}
		var reader io.Reader = cmd.InOrStdin()
		if input != "-" {
			f, e := os.Open(input)
			if e != nil {
				return e
			}
			defer f.Close()
			reader = f
		}
		report, err := checkWords(reader, policy.FromProfile(p))
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if reportPath != "" {
			if err = writePrivate(reportPath, data); err != nil {
				return err
			}
		} else {
			if _, err = cmd.OutOrStdout().Write(data); err != nil {
				return err
			}
		}
		if fail && report.Rejected > 0 {
			return fmt.Errorf("%d candidates violate policy", report.Rejected)
		}
		return nil
	}}
	cmd.Flags().StringVarP(&input, "input", "i", "-", "wordlist path, or - for stdin")
	cmd.Flags().StringVar(&policyPath, "policy", "", "standalone policy YAML (required)")
	cmd.Flags().StringVar(&reportPath, "report", "", "save aggregate JSON report to a new file (default stdout)")
	cmd.Flags().BoolVar(&fail, "fail-on-reject", false, "return nonzero when any candidate violates policy")
	_ = cmd.MarkFlagRequired("policy")
	return cmd
}
func newValidateCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{Use: "validate", Short: "Validate a YAML generation config without generating or creating output", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := profile.Load(path)
		if err != nil {
			return err
		}
		if cfg.Output.Mode == "rules" && policy.FromProfile(cfg.Policy).Active() {
			return fmt.Errorf("rules mode cannot enforce a password policy")
		}
		base := tokens.Build(cfg)
		if len(base) == 0 {
			return fmt.Errorf("profile contains no usable tokens")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Valid config: %d base tokens; mode=%s; budget=%d; policy=%t\n", len(base), strings.TrimSpace(cfg.Output.Mode), cfg.Output.Budget, policy.FromProfile(cfg.Policy).Active())
		return nil
	}}
	cmd.Flags().StringVarP(&path, "config", "c", "", "generation YAML config (required)")
	_ = cmd.MarkFlagRequired("config")
	return cmd
}
