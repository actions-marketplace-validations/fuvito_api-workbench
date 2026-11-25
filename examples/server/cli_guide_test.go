package main

import (
	"errors"
	"io/fs"
	"net/http/httptest"
	"os"
	osexec "os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// The detailed CLI guide (docs/guides/cli/) shows commands and what they print. This test runs
// them, with the built CLI against this server, so what a page says you will see stays true.
//
// Every ```bash block with a clrsig command has one of these comments on the line before it:
//
//	<!-- run: exit=1 contains="text" not="text" -->   run it; the last command must exit with 1
//	<!-- not run: the reason -->                      shown only (needs git, a remote service, ...)
//
// A ```text block after `<!-- output -->` is what the reader will see: each of its lines (except a
// line that is only …) must be in the output of the example before it. Times are not compared.
// Commands run in a copy of the repository's examples/ and docs/guides/cli/files/, one after the
// other, in a new folder for each block; every command but the last must exit with 0, or with the
// code given as first="N".

type guideExample struct {
	file     string
	line     int
	exit     int
	first    int
	contains []string
	not      []string
	commands [][]string
	output   [][]string // each block is checked on its own, in the order of its lines
}

var (
	reRunMarker = regexp.MustCompile(`^<!-- run: (.*) -->$`)
	reAttr      = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|(\d+))`)
	reDuration  = regexp.MustCompile(`\b\d+ ?ms\b`)
)

// words splits a command line as a shell would, for the quoting the guide uses.
func words(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	quote, has := rune(0), false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, has = r, true
		case r == ' ' || r == '\t':
			if has {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		case strings.ContainsRune("|<>&;`$", r):
			return nil, errors.New("a tested command cannot use the shell (" + string(r) + "): mark the block not run, or split it")
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	if quote != 0 {
		return nil, errors.New("a quote is not closed")
	}
	if has {
		out = append(out, cur.String())
	}
	return out, nil
}

func guideExamples(t *testing.T, file string) (examples []guideExample, unmarked int) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	previous := "" // the last line that is not blank
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "```") {
			if line != "" {
				previous = line
			}
			continue
		}
		lang, start := strings.TrimPrefix(line, "```"), i
		var body []string
		for i++; i < len(lines) && strings.TrimSpace(lines[i]) != "```"; i++ {
			body = append(body, lines[i])
		}
		switch {
		case lang == "text" && previous == "<!-- output -->":
			if len(examples) == 0 {
				t.Errorf("%s:%d: an output block with no example before it", file, start+1)
				break
			}
			examples[len(examples)-1].output = append(examples[len(examples)-1].output, body)
		case lang == "bash" && strings.Contains(strings.Join(body, "\n"), "clrsig "):
			m := reRunMarker.FindStringSubmatch(previous)
			if m == nil {
				if !strings.HasPrefix(previous, "<!-- not run: ") {
					t.Errorf("%s:%d: say how to test this block: <!-- run: exit=0 --> or <!-- not run: reason -->", file, start+1)
				}
				unmarked++
				break
			}
			ex := guideExample{file: file, line: start + 1, exit: -1}
			for _, a := range reAttr.FindAllStringSubmatch(m[1], -1) {
				value := a[2] + a[3]
				switch a[1] {
				case "exit", "first":
					n, err := strconv.Atoi(value)
					if err != nil {
						t.Fatalf("%s:%d: %s=%q", file, start+1, a[1], value)
					}
					if a[1] == "exit" {
						ex.exit = n
					} else {
						ex.first = n
					}
				case "contains":
					ex.contains = append(ex.contains, value)
				case "not":
					ex.not = append(ex.not, value)
				default:
					t.Errorf("%s:%d: unknown attribute %s", file, start+1, a[1])
				}
			}
			if ex.exit < 0 {
				t.Errorf("%s:%d: a run marker needs exit=", file, start+1)
			}
			joined := strings.ReplaceAll(strings.Join(body, "\n"), "\\\n", " ")
			for _, c := range strings.Split(joined, "\n") {
				c = strings.TrimSpace(c)
				if c == "" || strings.HasPrefix(c, "#") {
					continue
				}
				args, err := words(c)
				if err != nil || len(args) == 0 || args[0] != "clrsig" {
					t.Errorf("%s:%d: only clrsig commands can be run: %q %v", file, start+1, c, err)
					continue
				}
				ex.commands = append(ex.commands, args[1:])
			}
			examples = append(examples, ex)
		}
		previous = "```"
	}
	return examples, unmarked
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "clrsig")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := osexec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join("..", "..", "apps", "cli")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the CLI: %v\n%s", err, out)
	}
	return bin
}

// copyTree copies the files a reader has, pointing every address at the test server.
func copyTree(t *testing.T, from, to, url string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if ext := filepath.Ext(path); ext != ".yaml" && ext != ".json" {
			return nil
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(strings.ReplaceAll(string(data), "http://localhost:8080", url)), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDetailedGuideExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	pages, err := filepath.Glob(filepath.Join("..", "..", "docs", "guides", "cli", "*.md"))
	if err != nil || len(pages) == 0 {
		t.Fatalf("no guide pages: %v %v", pages, err)
	}
	var examples []guideExample
	for _, page := range pages {
		found, _ := guideExamples(t, page)
		examples = append(examples, found...)
	}
	if len(examples) < 10 {
		t.Fatalf("only %d examples were found, so little is checked", len(examples))
	}
	bin := buildCLI(t)

	for _, ex := range examples {
		t.Run(filepath.Base(ex.file)+":"+strconv.Itoa(ex.line), func(t *testing.T) {
			resetState()
			srv := httptest.NewServer(newMux())
			defer srv.Close()
			dir, home := t.TempDir(), t.TempDir()
			copyTree(t, "..", filepath.Join(dir, "examples"), srv.URL)
			copyTree(t, filepath.Join("..", "..", "docs", "guides", "cli", "files"), filepath.Join(dir, "docs", "guides", "cli", "files"), srv.URL)

			var all strings.Builder
			for i, args := range ex.commands {
				pointed := make([]string, len(args))
				for j, a := range args {
					pointed[j] = strings.ReplaceAll(a, "http://localhost:8080", srv.URL)
				}
				cmd := osexec.Command(bin, pointed...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "CLRSIG_CA_CERT=", "CLRSIG_CLIENT_CERT=", "CLRSIG_CLIENT_KEY=")
				out, err := cmd.CombinedOutput()
				code := 0
				var exit *osexec.ExitError
				if errors.As(err, &exit) {
					code = exit.ExitCode()
				} else if err != nil {
					t.Fatal(err)
				}
				text := strings.ReplaceAll(string(out), srv.URL, "http://localhost:8080")
				all.WriteString(text)
				want := ex.first
				if i == len(ex.commands)-1 {
					want = ex.exit
				}
				t.Logf("$ clrsig %s  (exit %d)\n%s", strings.Join(args, " "), code, text)
				if code != want {
					t.Errorf("%s:%d: clrsig %s exited %d, the guide says %d\n%s", ex.file, ex.line, strings.Join(args, " "), code, want, text)
				}
			}
			got := reDuration.ReplaceAllString(all.String(), "N ms")
			for _, want := range ex.contains {
				if !strings.Contains(got, reDuration.ReplaceAllString(want, "N ms")) {
					t.Errorf("%s:%d: the output does not contain %q\n%s", ex.file, ex.line, want, got)
				}
			}
			for _, bad := range ex.not {
				if strings.Contains(got, bad) {
					t.Errorf("%s:%d: the output must not contain %q\n%s", ex.file, ex.line, bad, got)
				}
			}
			var have []string
			for _, l := range strings.Split(got, "\n") {
				have = append(have, strings.TrimRight(l, " \r"))
			}
			for _, block := range ex.output {
				next := 0
				for _, line := range block {
					want := strings.TrimRight(reDuration.ReplaceAllString(line, "N ms"), " ")
					if want == "" || strings.TrimSpace(want) == "…" {
						continue
					}
					// in the order the block shows them
					for next < len(have) && have[next] != want {
						next++
					}
					if next == len(have) {
						t.Errorf("%s:%d: the page shows a line that is not in the output (or not in this order): %q\n%s", ex.file, ex.line, want, got)
						break
					}
					next++
				}
			}
		})
	}
}
