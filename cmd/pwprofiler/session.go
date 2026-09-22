package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/praneeth132006/Password-Profiler/internal/profile"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const maxInputBytes = 1 << 20
const maxInputLines = 2000

// A session is shared by the console, file CLI and browser. File contents stay local.
type session struct {
	Keywords []string
	Sources  []string
	Min      int
	Max      int
	Require  []string
	Budget   int
	Output   string
}

func newSession() *session {
	return &session{Min: 8, Max: 24, Require: []string{"upper", "lower", "digit", "special"}, Budget: 10000, Output: "passwords.txt"}
}

func readWords(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxInputBytes {
		return nil, fmt.Errorf("input exceeds 1 MiB")
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return nil, fmt.Errorf("input must be UTF-8 text")
	}
	var words []string
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.ContainsFunc(line, func(r rune) bool { return unicode.IsControl(r) && r != '\t' }) {
			return nil, fmt.Errorf("input contains a control character")
		}
		if len(line) > 256 {
			return nil, fmt.Errorf("input line exceeds 256 bytes")
		}
		words = append(words, line)
		if len(words) > maxInputLines {
			return nil, fmt.Errorf("input exceeds %d non-empty lines", maxInputLines)
		}
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("input has no usable lines")
	}
	return words, nil
}

func (s *session) add(r io.Reader, label string) error {
	words, err := readWords(r)
	if err != nil {
		return err
	}
	if len(s.Keywords)+len(words) > maxInputLines {
		return fmt.Errorf("session exceeds %d input lines", maxInputLines)
	}
	s.Keywords = append(s.Keywords, words...)
	s.Sources = append(s.Sources, fmt.Sprintf("%s (%d lines)", label, len(words)))
	return nil
}
func (s *session) addFile(path, label string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input must be a regular file")
	}
	return s.add(f, label+" : "+path)
}
func (s *session) config() (*profile.Config, error) {
	if s.Budget < 1 || s.Budget > 100000 {
		return nil, fmt.Errorf("budget must be between 1 and 100000")
	}
	if s.Min < 0 || s.Max < 1 || s.Max > 128 {
		return nil, fmt.Errorf("minimum must be nonnegative and maximum must be 1–128")
	}
	cfg := profile.Config{Profile: profile.Profile{Keywords: s.Keywords}, Rules: profile.Rules{Case: []string{"lower", "capitalize", "upper"}, Leet: "partial", MaxCombine: 1, Depth: 2, Affixes: profile.Affixes{Suffixes: []string{"1!", "123!", strconv.Itoa(time.Now().Year()) + "!", "!", "123"}}}, Policy: profile.Policy{MinLen: s.Min, MaxLen: s.Max, Require: s.Require}, Output: profile.Output{Mode: "wordlist", File: s.Output, Budget: s.Budget, Dedupe: true}}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return profile.Parse(data)
}
func (s *session) run(cmd *cobra.Command) error {
	cfg, err := s.config()
	if err != nil {
		return err
	}
	if strings.TrimSpace(s.Output) == "" {
		return fmt.Errorf("output path is required")
	}
	return runGenerate(cmd, cfg, genOpts{workers: 1, dedup: "exact"})
}

func newFilesCmd() *cobra.Command {
	s := newSession()
	var files, emails, names, companies []string
	cmd := &cobra.Command{Use: "files", Short: "Generate passwords.txt from one or more text files", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		for _, group := range []struct {
			label string
			paths []string
		}{{"Keywords", files}, {"Business email found", emails}, {"Names", names}, {"Companies", companies}} {
			for _, path := range group.paths {
				if err := s.addFile(path, group.label); err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
			}
		}
		return s.run(cmd)
	}}
	cmd.Flags().StringArrayVarP(&files, "input", "i", nil, "text file, one entry per line (repeatable)")
	cmd.Flags().StringArrayVar(&emails, "emails", nil, "business email text file (repeatable)")
	cmd.Flags().StringArrayVar(&names, "names", nil, "names text file (repeatable)")
	cmd.Flags().StringArrayVar(&companies, "companies", nil, "company text file (repeatable)")
	cmd.Flags().IntVar(&s.Min, "min-length", s.Min, "minimum password length in characters")
	cmd.Flags().IntVar(&s.Max, "max-length", s.Max, "maximum password length in characters")
	cmd.Flags().StringSliceVar(&s.Require, "require", s.Require, "required classes: upper,lower,digit,special; empty disables")
	cmd.Flags().IntVar(&s.Budget, "budget", s.Budget, "maximum output candidates (1–100000)")
	cmd.Flags().StringVarP(&s.Output, "output", "o", s.Output, "destination text file (must not exist)")
	return cmd
}

const consoleHelp = `Commands:
  add emails /path/to/email.txt     Business email found : file
  add names /path/to/names.txt      Names file
  add companies /path/to/orgs.txt   Companies file
  add keywords /path/to/words.txt   Any other text file
  set min 8                        Minimum length
  set max 24                       Maximum length
  set require upper,lower,digit,special  (or none)
  set budget 10000                 Maximum candidates (up to 100000)
  set output /path/passwords.txt    New output file
  show options                     Review files and password policy
  run                              Generate the text wordlist
  reset                            Clear session and restore defaults
  help / exit
Paths may contain spaces; optional surrounding quotes are supported.
`

func newConsoleCmd() *cobra.Command {
	return &cobra.Command{Use: "console", Short: "Start an interactive audit workspace", Args: cobra.NoArgs, RunE: runConsole}
}
func runConsole(cmd *cobra.Command, _ []string) error {
	s := newSession()
	out := cmd.OutOrStdout()
	fmt.Fprint(out, "\n  PASSWORD PROFILER  /  Local audit workspace\n  Type help for commands. Add files, set policy, then run.\n\n")
	scanner := bufio.NewScanner(cmd.InOrStdin())
	scanner.Buffer(make([]byte, 4096), maxInputBytes)
	for {
		fmt.Fprint(out, "pwprofiler > ")
		if !scanner.Scan() {
			return scanner.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		verb, rest, _ := strings.Cut(line, " ")
		rest = strings.TrimSpace(rest)
		var err error
		switch strings.ToLower(verb) {
		case "":
			continue
		case "help", "?":
			fmt.Fprint(out, consoleHelp)
		case "exit", "quit":
			return nil
		case "reset":
			s = newSession()
			fmt.Fprintln(out, "[+] Session reset")
		case "show":
			if rest != "options" {
				err = fmt.Errorf("use show options")
				break
			}
			fmt.Fprintf(out, "\nSources (%d lines):\n", len(s.Keywords))
			for _, source := range s.Sources {
				fmt.Fprintln(out, "  "+source)
			}
			fmt.Fprintf(out, "Policy: %d–%d characters; require %s\nBudget: %d\nOutput: %s\n\n", s.Min, s.Max, strings.Join(s.Require, ","), s.Budget, s.Output)
		case "add":
			kind, path, _ := strings.Cut(rest, " ")
			path = strings.Trim(strings.TrimSpace(path), "\"'")
			labels := map[string]string{"emails": "Business email found", "names": "Names found", "companies": "Companies found", "keywords": "Keywords found"}
			label, ok := labels[kind]
			if !ok || path == "" {
				err = fmt.Errorf("use add emails|names|companies|keywords /path/to/file.txt")
				break
			}
			err = s.addFile(path, label)
			if err == nil {
				fmt.Fprintln(out, "[+] "+s.Sources[len(s.Sources)-1])
			}
		case "set":
			key, value, _ := strings.Cut(rest, " ")
			value = strings.Trim(strings.TrimSpace(value), "\"'")
			switch key {
			case "output":
				s.Output = value
			case "require":
				s.Require = nil
				if value != "none" && value != "" {
					s.Require = strings.Split(value, ",")
				}
			case "min", "max", "budget":
				var n int
				n, err = strconv.Atoi(value)
				if err == nil {
					switch key {
					case "min":
						s.Min = n
					case "max":
						s.Max = n
					case "budget":
						s.Budget = n
					}
				}
			default:
				err = fmt.Errorf("unknown option %q; type help", key)
			}
		case "run":
			err = s.run(cmd)
			if err == nil {
				fmt.Fprintln(out, "[+] Saved : "+s.Output)
			}
		default:
			err = fmt.Errorf("unknown command %q; type help", verb)
		}
		if err != nil {
			fmt.Fprintln(out, "[-] "+err.Error())
		}
	}
}
