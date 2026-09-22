package main

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/praneeth132006/Password-Profiler/internal/policy"
)

func assertWordlist(t *testing.T, data string) {
	t.Helper()
	if strings.TrimSpace(data) == "" {
		t.Fatal("empty wordlist")
	}
	filter := policy.New(policy.Policy{MinLen: 8, MaxLen: 24, Require: []string{"upper", "lower", "digit", "special"}})
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		if !filter.Allow(line) {
			t.Fatalf("candidate fails policy: %q", line)
		}
		if seen[line] {
			t.Fatalf("duplicate %q", line)
		}
		seen[line] = true
	}
}
func TestFilesAndConsole(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "business emails.txt")
	if err := os.WriteFile(input, []byte("alex@northstar.example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"files", "console"} {
		t.Run(mode, func(t *testing.T) {
			output := filepath.Join(dir, mode+".txt")
			cmd := newRootCmd()
			var log bytes.Buffer
			cmd.SetOut(&log)
			cmd.SetErr(&log)
			if mode == "files" {
				cmd.SetArgs([]string{"files", "--emails", input, "--output", output, "--budget", "50"})
			} else {
				cmd.SetArgs([]string{"console"})
				cmd.SetIn(strings.NewReader("add emails " + input + "\nset output " + output + "\nset budget 50\nshow options\nrun\nexit\n"))
			}
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			assertWordlist(t, string(data))
			if bytes.Count(data, []byte{'\n'}) > 50 {
				t.Fatal("budget exceeded")
			}
			if mode == "console" && !strings.Contains(log.String(), "Business email found : "+input) {
				t.Fatal(log.String())
			}
			s := newSession()
			s.Keywords = []string{"northstar"}
			s.Output = output
			if err = s.run(cmd); err == nil {
				t.Fatal("overwrote existing output")
			}
			after, _ := os.ReadFile(output)
			if !bytes.Equal(data, after) {
				t.Fatal("existing output changed")
			}
		})
	}
}
func TestInputValidation(t *testing.T) {
	for _, input := range []string{"", "# comment\n", "bad\x00text", strings.Repeat("x", 257), strings.Repeat("a\n", 2001), strings.Repeat("x", maxInputBytes+1)} {
		if _, err := readWords(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted invalid input (%d bytes)", len(input))
		}
	}
	words, err := readWords(strings.NewReader("\ufeffalpha\r\n# comment\r\n\r\nbeta\r\n"))
	if err != nil || len(words) != 2 {
		t.Fatalf("words=%v err=%v", words, err)
	}
}
func TestNoMatchAndCancellationRemoveOutput(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		s := newSession()
		s.Keywords = []string{"northstar"}
		s.Output = filepath.Join(t.TempDir(), "passwords.txt")
		if !cancel {
			s.Min = 100
			s.Max = 101
		}
		cmd := newRootCmd()
		ctx, stop := context.WithCancel(context.Background())
		if cancel {
			stop()
		}
		defer stop()
		cmd.SetContext(ctx)
		if err := s.run(cmd); err == nil {
			t.Fatal("expected failure")
		}
		if _, err := os.Stat(s.Output); !os.IsNotExist(err) {
			t.Fatal("failed generation left a file")
		}
	}
}
func request(t *testing.T, words string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"min": "8", "max": "24", "budget": "50", "words": words} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"upper", "lower", "digit", "special"} {
		_ = writer.WriteField("require", v)
	}
	part, err := writer.CreateFormFile("emails", "emails.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("alex@northstar.example\n"))
	_ = writer.Close()
	req := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/generate", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Pwprofiler", "local")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	return req
}
func TestBrowserWorkflow(t *testing.T) {
	handler := webHandler("127.0.0.1:8080")
	t.Run("page", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:8080/", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Build a more focused wordlist") {
			t.Fatal(w.Code)
		}
	})
	t.Run("download", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request(t, "Riverstone"))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		assertWordlist(t, w.Body.String())
		if !strings.Contains(w.Header().Get("Content-Disposition"), "passwords.txt") {
			t.Fatal("missing download header")
		}
	})
	for _, kind := range []string{"host", "origin", "header", "invalid input"} {
		t.Run(kind, func(t *testing.T) {
			req := request(t, "Riverstone")
			switch kind {
			case "host":
				req.Host = "attacker.example"
			case "origin":
				req.Header.Set("Origin", "https://attacker.example")
			case "header":
				req.Header.Del("X-Pwprofiler")
			case "invalid input":
				req = request(t, strings.Repeat("a", 257))
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code < 400 {
				t.Fatal("request accepted")
			}
		})
	}
}

func TestRulesPolicyAndOutputSafety(t *testing.T) {
	s := newSession()
	s.Keywords = []string{"Northstar"}
	s.Output = filepath.Join(t.TempDir(), "words.txt")
	cfg, err := s.config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.Mode = "rules"
	cmd := newRootCmd()
	if err = runGenerate(cmd, cfg, genOpts{}); err == nil {
		t.Fatal("rules accepted a policy")
	}
	cfg.Policy.MinLen = 0
	cfg.Policy.MaxLen = 0
	cfg.Policy.Require = nil
	rulePath := deriveRulePath(s.Output)
	if err = os.WriteFile(rulePath, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = runGenerate(cmd, cfg, genOpts{}); err == nil {
		t.Fatal("overwrote rule file")
	}
	data, _ := os.ReadFile(rulePath)
	if string(data) != "existing" {
		t.Fatal("existing rule file changed")
	}
	if _, err = os.Stat(s.Output); !os.IsNotExist(err) {
		t.Fatal("failed rule export left wordlist")
	}
}
