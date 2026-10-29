// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grep

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// options are grep's flags.
type options struct {
	ignoreCase, invert, withNumber, regex, recursive bool
	filesOnly, count, onlyMatching, quiet, word      bool
	withName, noName                                 bool
}

// MaxDepth bounds the directories -r descends into.
const MaxDepth = 64

// Run searches each named file (standard input when there is none, or for
// "-") for PATTERN and prints matching lines.
//
// By default PATTERN is a literal substring (-F says so explicitly); with -E
// it is compiled by the pure-Go go-ruby-regexp engine (Onigmo subset:
// alternation, anchors, classes, quantifiers). Flags: -i case-insensitive,
// -v invert, -n line numbers, -c count, -l names of matching files, -o only
// the matching parts, -q quiet (status only), -w whole words, -r/-R
// recursive (directories walked, "." without operands), -H/-h force/omit
// the "FILE:" prefix (default: on with several files or -r). Short flags
// combine (-rn).
//
// Under -E, -i is the inline (?i) flag, so case-folding is Unicode-correct;
// on the substring path it compares lower-cased text.
//
// Exit codes: Ok (0) if a line was selected, Fail (1) if none, Usage (2) on
// argument errors (an invalid -E pattern included), or when a file could
// not be read and nothing was selected, as POSIX grep.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var o options
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			pos = append(pos, a)
			continue
		}
		for _, c := range a[1:] {
			switch c {
			case 'i':
				o.ignoreCase = true
			case 'v':
				o.invert = true
			case 'n':
				o.withNumber = true
			case 'E':
				o.regex = true
			case 'F':
				o.regex = false
			case 'r', 'R':
				o.recursive = true
			case 'l':
				o.filesOnly = true
			case 'c':
				o.count = true
			case 'o':
				o.onlyMatching = true
			case 'q':
				o.quiet = true
			case 'w':
				o.word = true
			case 'H':
				o.withName, o.noName = true, false
			case 'h':
				o.noName, o.withName = true, false
			default:
				fmt.Fprintf(env.Stderr, "grep: invalid option -- '%c'\n", c)
				return exit.Usage
			}
		}
	}
	if len(pos) < 1 {
		fmt.Fprintln(env.Stderr, "grep: usage: grep [-EFivnclqowrRHh] PATTERN [FILE...]")
		return exit.Usage
	}
	pattern, files := pos[0], pos[1:]
	if len(files) == 0 {
		if o.recursive {
			files = []string{"."}
		} else {
			files = []string{fsx.StdinOperand}
		}
	}
	matcher, err := buildMatcher(pattern, o.regex, o.ignoreCase, o.word)
	if err != nil {
		fmt.Fprintf(env.Stderr, "grep: %s\n", err.Error())
		return exit.Usage
	}
	if o.recursive {
		files = expand(env, files)
	}
	showName := (len(files) > 1 || o.recursive) && !o.noName || o.withName

	selected, failed := false, false
	for _, p := range files {
		data, err := env.ReadOperand(p)
		if err != nil {
			if !o.quiet {
				fmt.Fprintf(env.Stderr, "grep: %s: %s\n", p, prettyErr(err))
			}
			failed = true
			continue
		}
		name := p
		if p == fsx.StdinOperand {
			name = "(standard input)"
		}
		if search(env, &o, matcher, string(data), name, showName) > 0 {
			selected = true
			if o.quiet {
				return exit.Ok
			}
		}
	}
	switch {
	case selected:
		return exit.Ok
	case failed:
		return exit.Usage
	}
	return exit.Fail
}

// search prints the selected lines of text and returns how many there were.
func search(env *fsx.Env, o *options, matcher func(string) [][2]int, text, name string, showName bool) int {
	selected, lineNum := 0, 0
	for text != "" {
		lineNum++
		i := strings.IndexByte(text, '\n')
		var line string
		if i < 0 {
			line, text = text, ""
		} else {
			line, text = text[:i], text[i+1:]
		}
		spans := matcher(line)
		if (len(spans) > 0) == o.invert {
			continue
		}
		selected++
		switch {
		case o.filesOnly:
			if !o.quiet {
				fmt.Fprintln(env.Stdout, name)
			}
			return selected
		case o.quiet || o.count:
			continue
		}
		prefix := ""
		if showName {
			prefix += name + ":"
		}
		if o.withNumber {
			prefix += fmt.Sprintf("%d:", lineNum)
		}
		if o.onlyMatching {
			// GNU: -o -v prints nothing; empty matches are not printed.
			for _, s := range spans {
				if !o.invert && s[1] > s[0] {
					fmt.Fprintf(env.Stdout, "%s%s\n", prefix, line[s[0]:s[1]])
				}
			}
			continue
		}
		fmt.Fprintf(env.Stdout, "%s%s\n", prefix, line)
	}
	if o.count && !o.quiet && !o.filesOnly {
		if showName {
			fmt.Fprintf(env.Stdout, "%s:%d\n", name, selected)
		} else {
			fmt.Fprintf(env.Stdout, "%d\n", selected)
		}
	}
	return selected
}

// expand replaces each directory operand by the files under it, sorted,
// recursively (MaxDepth levels); "-" and files stay as they are.
func expand(env *fsx.Env, ops []string) []string {
	var out []string
	var walk func(shown, abs string, depth int)
	walk = func(shown, abs string, depth int) {
		st, err := env.FS.Stat(abs)
		if err != nil || !st.IsDir {
			out = append(out, shown)
			return
		}
		if depth >= MaxDepth {
			return
		}
		kids, err := env.FS.ReadDir(abs)
		if err != nil {
			out = append(out, shown)
			return
		}
		sort.Slice(kids, func(i, j int) bool { return kids[i].Name < kids[j].Name })
		for _, k := range kids {
			walk(strings.TrimSuffix(shown, "/")+"/"+k.Name, fsx.Join(abs, k.Name), depth+1)
		}
	}
	for _, op := range ops {
		if op == fsx.StdinOperand {
			out = append(out, op)
			continue
		}
		walk(op, fsx.Resolve(env.Cwd, op), 0)
	}
	return out
}

// isWord reports whether b is a word byte ([A-Za-z0-9_]).
func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// buildMatcher returns a function giving the non-overlapping match spans
// of a line, and a compile-time error (only with -E and a malformed
// pattern). With word, a match must not touch a word byte on either side.
func buildMatcher(pattern string, regex, ignoreCase, word bool) (func(string) [][2]int, error) {
	if regex {
		src := pattern
		if word {
			src = `(?<!\w)(?:` + src + `)(?!\w)`
		}
		if ignoreCase {
			// (?i) is Onigmo's inline case-insensitive flag and handles full
			// Unicode case-folding -- the correct vehicle for -i under -E.
			src = "(?i)" + src
		}
		re, err := onigmo.Compile(src)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %s", pattern, err.Error())
		}
		return func(line string) [][2]int {
			var spans [][2]int
			for off := 0; off <= len(line); {
				m := re.Match(line[off:])
				if m == nil {
					break
				}
				b, e := off+m.Begin(0), off+m.End(0)
				spans = append(spans, [2]int{b, e})
				if e == b {
					e++ // step past an empty match
				}
				off = e
			}
			return spans
		}, nil
	}
	needle := pattern
	if ignoreCase {
		needle = strings.ToLower(pattern)
	}
	return func(line string) [][2]int {
		hay := line
		if ignoreCase {
			hay = strings.ToLower(line)
		}
		if needle == "" {
			return [][2]int{{0, 0}}
		}
		var spans [][2]int
		for off := 0; off+len(needle) <= len(hay); {
			i := strings.Index(hay[off:], needle)
			if i < 0 {
				break
			}
			b, e := off+i, off+i+len(needle)
			if !word || (b == 0 || !isWord(hay[b-1])) && (e == len(hay) || !isWord(hay[e])) {
				spans = append(spans, [2]int{b, e})
				off = e
				continue
			}
			off = b + 1
		}
		return spans
	}, nil
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
