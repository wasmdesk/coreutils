// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grep

import (
	"errors"
	"fmt"
	"strings"

	onigmo "github.com/go-ruby-regexp/regexp"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run searches each named file for PATTERN and prints matching lines. By
// default PATTERN is a literal substring (POSIX-fixed semantics); with -E the
// pattern is compiled by the pure-Go go-ruby-regexp engine (Onigmo subset),
// enabling alternation, anchors, classes, quantifiers, etc. Flags: -i
// (case-insensitive), -v (invert match), -n (prefix line numbers), -E
// (extended-regex mode). With multiple files, each match line is prefixed
// with "FILE:".
//
// Under -E, -i is honoured by prepending the inline (?i) flag to the user's
// pattern -- this hands case-folding to the regex engine and so works
// correctly for non-ASCII letters (é/É, σ/Σ, ...) that a naive ToLower would
// botch. Under the substring path (-E absent), -i keeps its pre-existing
// ToLower-both-sides behaviour for backward compatibility.
//
// Exit codes: Ok (0) if any match found, Fail (1) otherwise, Usage (2) on
// argument errors -- including an invalid -E pattern. Matches the `grep`
// shell-script convention.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	ignoreCase, invert, withNumber, regex := false, false, false, false
	var pos []string
	for _, a := range args {
		switch a {
		case "-i":
			ignoreCase = true
		case "-v":
			invert = true
		case "-n":
			withNumber = true
		case "-E":
			regex = true
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) < 2 {
		fmt.Fprintln(env.Stderr, "grep: usage: grep [-E] [-i] [-v] [-n] PATTERN FILE...")
		return exit.Usage
	}
	pattern, files := pos[0], pos[1:]

	matcher, err := buildMatcher(pattern, regex, ignoreCase)
	if err != nil {
		fmt.Fprintf(env.Stderr, "grep: %s\n", err.Error())
		return exit.Usage
	}

	rc := exit.Fail
	for _, p := range files {
		abs := fsx.Resolve(env.Cwd, p)
		data, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "grep: %s: %s\n", p, prettyErr(err))
			continue
		}
		text := string(data)
		lineNum := 0
		// Iterate lines without losing the boundary distinction.
		for text != "" {
			lineNum++
			i := strings.IndexByte(text, '\n')
			var line string
			if i < 0 {
				line = text
				text = ""
			} else {
				line = text[:i]
				text = text[i+1:]
			}
			match := matcher(line)
			if invert {
				match = !match
			}
			if !match {
				continue
			}
			rc = exit.Ok
			printMatch(env.Stdout, line, lineNum, p, len(files) > 1, withNumber)
		}
	}
	return rc
}

// buildMatcher returns a per-line predicate plus a compile-time error (only
// non-nil when -E is set and the pattern is malformed). Split out so the
// hot loop in Run does not branch on (regex, ignoreCase) for every line.
func buildMatcher(pattern string, regex, ignoreCase bool) (func(string) bool, error) {
	if regex {
		src := pattern
		if ignoreCase {
			// (?i) is Onigmo's inline case-insensitive flag and handles full
			// Unicode case-folding -- the correct vehicle for -i under -E.
			src = "(?i)" + src
		}
		re, err := onigmo.Compile(src)
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %s", pattern, err.Error())
		}
		return re.MatchString, nil
	}
	if ignoreCase {
		lowered := strings.ToLower(pattern)
		return func(line string) bool {
			return strings.Contains(strings.ToLower(line), lowered)
		}, nil
	}
	return func(line string) bool {
		return strings.Contains(line, pattern)
	}, nil
}

func printMatch(w writer, line string, n int, path string, withPath, withNumber bool) {
	if withPath {
		fmt.Fprintf(w, "%s:", path)
	}
	if withNumber {
		fmt.Fprintf(w, "%d:", n)
	}
	fmt.Fprintln(w, line)
}

type writer interface{ Write([]byte) (int, error) }

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
