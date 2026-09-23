package combine

import (
	"fmt"
	"testing"
)

func BenchmarkSmallLimit(b *testing.B) {
	toks := make([]string, 500)
	for i := range toks {
		toks[i] = fmt.Sprintf("token%04d", i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := Generate(toks, Config{MaxCombine: 3, Limit: 510})
		if len(res.Tokens) != 510 {
			b.Fatal(len(res.Tokens))
		}
	}
}
