package main

import (
	"io"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/dedup"
	"github.com/praneeth132006/Password-Profiler/internal/mutate"
	"github.com/praneeth132006/Password-Profiler/internal/output"
	"github.com/praneeth132006/Password-Profiler/internal/policy"
)

func BenchmarkGenerationBudget(b *testing.B) {
	eng := mutate.NewEngine(mutate.Config{Cases: []string{"lower", "capitalize", "upper"}, Leet: "full", Append: []string{"1!", "123!", "2026!", "!", "123"}, Structural: true, Depth: 3})
	pol := policy.New(policy.Policy{MinLen: 8, MaxLen: 24, Require: []string{"upper", "lower", "digit", "special"}})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := output.New(io.Discard, dedup.NewExact(), 50)
		if _, err := expandSequential([]string{"northstar", "riverstone"}, eng, pol, w); err != nil {
			b.Fatal(err)
		}
		stats, err := w.Flush()
		if err != nil || stats.Emitted != 50 {
			b.Fatalf("stats=%+v err=%v", stats, err)
		}
	}
}
