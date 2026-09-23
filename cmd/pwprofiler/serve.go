package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

//go:embed web/index.html
var indexHTML []byte

//go:embed web/styles.css
var styleCSS []byte

//go:embed web/app.js
var appJS []byte

func newServeCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{Use: "serve", Short: "Open the local browser UI at http://127.0.0.1:8080", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if port < 0 || port > 65535 {
			return fmt.Errorf("port must be 0–65535")
		}
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return err
		}
		host := listener.Addr().String()
		server := &http.Server{Handler: webHandler(host), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 2 * time.Minute, IdleTimeout: 60 * time.Second}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		finished := make(chan struct{})
		defer close(finished)
		go func() {
			select {
			case <-ctx.Done():
				shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdown)
			case <-finished:
			}
		}()
		fmt.Fprintf(cmd.OutOrStdout(), "Password Profiler UI: http://%s\nPress Ctrl+C to stop.\n", host)
		err = server.Serve(listener)
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}}
	cmd.Flags().IntVar(&port, "port", 8080, "localhost port (0 chooses an available port)")
	return cmd
}

func webHandler(host string) http.Handler {
	busy := make(chan struct{}, 1)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.Host != host {
			http.Error(w, "Invalid host", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet {
			var data []byte
			var contentType string
			switch r.URL.Path {
			case "/":
				data = indexHTML
				contentType = "text/html; charset=utf-8"
			case "/styles.css":
				data = styleCSS
				contentType = "text/css; charset=utf-8"
			case "/app.js":
				data = appJS
				contentType = "text/javascript; charset=utf-8"
			}
			if data != nil {
				w.Header().Set("Content-Type", contentType)
				_, _ = w.Write(data)
				return
			}
		}
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST")
			http.Error(w, "Use POST", 405)
			return
		}
		if r.Header.Get("X-Pwprofiler") != "local" || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+host) {
			http.Error(w, "Use the local application to generate a wordlist", 403)
			return
		}
		select {
		case busy <- struct{}{}:
			defer func() { <-busy }()
		default:
			http.Error(w, "A generation is running. Please try again shortly.", 429)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			http.Error(w, "Invalid upload or request exceeds 4 MiB", 400)
			return
		}
		defer r.MultipartForm.RemoveAll()
		s := newSession()
		s.Output = ""
		var err error
		for key, dest := range map[string]*int{"min": &s.Min, "max": &s.Max, "budget": &s.Budget} {
			*dest, err = strconv.Atoi(r.FormValue(key))
			if err != nil {
				http.Error(w, "Invalid "+key, 400)
				return
			}
		}
		for key, dest := range map[string]*int{"max_bytes": &s.MaxBytes, "max_repeat": &s.MaxRepeat} {
			if value := r.FormValue(key); value != "" {
				*dest, err = strconv.Atoi(value)
				if err != nil {
					http.Error(w, "Invalid "+key, 400)
					return
				}
			}
		}
		s.Forbidden = r.FormValue("forbidden")
		for _, word := range strings.Split(strings.ReplaceAll(r.FormValue("blocklist"), "\r\n", "\n"), "\n") {
			if word != "" {
				s.Blocklist = append(s.Blocklist, word)
			}
		}
		s.Require = r.MultipartForm.Value["require"]
		for _, kind := range []string{"emails", "names", "companies", "keywords"} {
			for _, header := range r.MultipartForm.File[kind] {
				f, e := header.Open()
				if e != nil {
					http.Error(w, "Cannot read upload", 400)
					return
				}
				e = s.add(f, header.Filename)
				_ = f.Close()
				if e != nil {
					http.Error(w, e.Error(), 400)
					return
				}
			}
		}
		if text := strings.TrimSpace(r.FormValue("words")); text != "" {
			if err = s.add(strings.NewReader(text), "Manual entries"); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
		}
		cfg, err := s.config()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var result, stats bytes.Buffer
		cmd := &cobra.Command{}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		cmd.SetContext(ctx)
		cmd.SetOut(&result)
		cmd.SetErr(&stats)
		report := &generationReport{}
		if err = runWordlist(cmd, cfg, genOpts{workers: 1, dedup: "exact", report: report}); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		if result.Len() == 0 {
			http.Error(w, "No candidates match this policy. Add longer input words or adjust the policy.", 422)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="passwords.txt"`)
		count := bytes.Count(result.Bytes(), []byte{'\n'})
		w.Header().Set("X-Candidate-Count", strconv.Itoa(count))
		w.Header().Set("X-Search-Limited", strconv.FormatBool(report.SearchLimited))
		w.Header().Set("X-Limit-Reached", strconv.FormatBool(count >= s.Budget))
		_, _ = w.Write(result.Bytes())
	})
}
