package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/dedup"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
)

func TestStreamingWorkersEquivalent(t *testing.T) {
	var first []string
	for _, workers := range []int{1, 2, 4} {
		var sink bytes.Buffer
		w := output.New(&sink, dedup.NewExact(), 100000)
		eng := mutate.NewEngine(mutate.Config{Cases: []string{"lower", "capitalize"}, Leet: "partial", Append: []string{"1!", "123"}, Depth: 2, Structural: true})
		_, err := expandStream(context.Background(), sliceSource([]string{"northstar", "riverstone"}), eng, policy.New(policy.Policy{MinLen: 8}), w, workers)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Flush(); err != nil {
			t.Fatal(err)
		}
		values := strings.Split(strings.TrimSpace(sink.String()), "\n")
		sort.Strings(values)
		if first == nil {
			first = values
		} else if !reflect.DeepEqual(values, first) {
			t.Fatal("worker counts produced different candidate sets")
		}
	}
}
func TestCancellationReachesWorkers(t *testing.T) {
	for _, workers := range []int{1, 4} {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		w := output.New(io.Discard, nil, 100)
		_, err := expandStream(ctx, sliceSource([]string{strings.Repeat("a", 100)}), mutate.NewEngine(mutate.Config{Leet: "full", Depth: 8}), policy.New(policy.Policy{}), w, workers)
		if err != context.Canceled {
			t.Fatalf("workers=%d err=%v", workers, err)
		}
	}
}
func TestOutputBudgetDoesNotTruncateInputSearch(t *testing.T) {
	cfg, err := profile.Parse([]byte(`profile:
  keywords: [a, b, c, Northstar]
rules:
  case: [capitalize]
  affixes:
    suffixes: ['1!']
policy:
  min_len: 8
  require: [upper, lower, digit, special]
output:
  budget: 1
  dedupe: true
`))
	if err != nil {
		t.Fatal(err)
	}
	var result, logs bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&result)
	cmd.SetErr(&logs)
	if err = runWordlist(cmd, cfg, genOpts{workers: 1, dedup: "exact"}); err != nil {
		t.Fatal(err)
	}
	if result.String() != "Northstar1!\n" {
		t.Fatal(result.String())
	}
}
func TestSearchLimitsAreReported(t *testing.T) {
	cfg, err := profile.Parse([]byte("profile:\n  keywords: [northstar]\nrules:\n  max_variants: 1\noutput:\n  budget: 50\n"))
	if err != nil {
		t.Fatal(err)
	}
	var out, logs bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&logs)
	report := &generationReport{}
	if err = runWordlist(cmd, cfg, genOpts{workers: 1, report: report}); err != nil {
		t.Fatal(err)
	}
	if !report.SearchLimited || !strings.Contains(logs.String(), "coverage is incomplete") {
		t.Fatal(logs.String())
	}
}
func TestCancelledGenerationRemovesPartialFile(t *testing.T) {
	s := newSession()
	s.Keywords = []string{"northstar"}
	s.Output = filepath.Join(t.TempDir(), "out.txt")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := newRootCmd()
	cmd.SetContext(ctx)
	if err := s.run(cmd); err != context.Canceled {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Output); !os.IsNotExist(err) {
		t.Fatal("partial output remains")
	}
}

func TestPolicyRejectDoesNotPruneRepairableIntermediate(t *testing.T) {
	var sink bytes.Buffer
	w := output.New(&sink, dedup.NewExact(), 1000)
	eng := mutate.NewEngine(mutate.Config{Structural: true, Depth: 1})
	_, err := expandStream(context.Background(), sliceSource([]string{"abcdefg"}), eng, policy.New(policy.Policy{MinLen: 6, MaxLen: 6}), w, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sink.String(), "abcdef\n") {
		t.Fatal("rejected long seed incorrectly pruned a valid truncation")
	}
}

func TestWorkerBudgetStopsUpstreamSearch(t *testing.T) {
	for _, workers := range []int{1, 4} {
		var sink bytes.Buffer
		w := output.New(&sink, dedup.NewExact(), 3)
		eng := mutate.NewEngine(mutate.Config{Leet: "full", Structural: true, Depth: 8})
		stats, err := expandStream(context.Background(), sliceSource([]string{strings.Repeat("a", 100), strings.Repeat("s", 100)}), eng, policy.New(policy.Policy{}), w, workers)
		if err != nil {
			t.Fatal(err)
		}
		summary, err := w.Flush()
		if err != nil || summary.Emitted != 3 || stats.Attempts >= 10000 {
			t.Fatalf("workers=%d stats=%+v output=%+v err=%v", workers, stats, summary, err)
		}
	}
}

type failingSink struct{}

var errTestSink = errors.New("test sink failed")

func (failingSink) Write([]byte) (int, error) { return 0, errTestSink }

func TestWorkerWriteFailureCancelsAndDrains(t *testing.T) {
	eng := mutate.NewEngine(mutate.Config{Append: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, Depth: 3})
	w := output.New(failingSink{}, dedup.NewExact(), 10000)
	_, err := expandStream(context.Background(), sliceSource([]string{strings.Repeat("ab", 1000), strings.Repeat("cd", 1000)}), eng, policy.New(policy.Policy{}), w, 4)
	if !errors.Is(err, errTestSink) {
		t.Fatalf("got %v", err)
	}
}
