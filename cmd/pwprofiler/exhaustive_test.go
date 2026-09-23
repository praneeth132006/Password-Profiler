package main

import (
	"bytes"
	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExhaustiveEndToEnd(t *testing.T) {
	cfg, err := profile.Parse([]byte(`profile:
  keywords: [xx, yy]
rules:
  exhaustive: true
  repeat_tokens: true
  max_combine: 2
  depth: 1
  max_variants: 1
  max_attempts: 1
  max_candidate_bytes: 1
  combine_limit: 1
  combine_attempts: 1
output:
  budget: 1
  dedupe: false
`))
	if err != nil {
		t.Fatal(err)
	}
	var out, logs bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&logs)
	report := &generationReport{}
	if err = runWordlist(cmd, cfg, genOpts{workers: 2, dedup: "bloom", report: report}); err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, s := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if got[s] {
			t.Fatal("duplicate", s)
		}
		got[s] = true
	}
	for _, s := range []string{"xx", "yy", "xxyy", "yyxx", "xxxx", "yyyy", "xxyyxxyy"} {
		if !got[s] {
			t.Fatal("missing", s)
		}
	}
	if report.SearchLimited || !strings.Contains(logs.String(), "dedup=exact") || !strings.Contains(logs.String(), "completed all configured") {
		t.Fatal(logs.String())
	}
}
func TestBrowserExhaustiveConfigExport(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	for k, v := range map[string]string{"words": "Northstar\nRiverstone", "min": "8", "max": "24", "budget": "100000", "max_combine": "3", "depth": "2", "leet": "full", "leet_cap": "-1", "repeat_tokens": "on"} {
		if err := form.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	form.Close()
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/config", &body)
	r.Header.Set("Content-Type", form.FormDataContentType())
	r.Header.Set("X-Pwprofiler", "local")
	w := httptest.NewRecorder()
	webHandler("127.0.0.1:8080").ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	cfg, err := profile.Parse(w.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Rules.Exhaustive || !cfg.Rules.RepeatTokens || cfg.Rules.MaxCombine != 3 || cfg.Rules.LeetCap != -1 || cfg.Output.File != "passwords.txt" {
		t.Fatalf("%+v", cfg)
	}
}
