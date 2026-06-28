// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grep

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run searches for a SUBSTRING (NOT a regex in v0; see README) in each named
// file and prints matching lines. Flags: -i (case-insensitive), -v (invert
// match), -n (prefix line numbers). With multiple files, each match line is
// prefixed with "FILE:". v0 deliberately ships substring-only matching
// because go-ruby-regexp pulls in the full Onigmo engine; the next iteration
// will swap in a regex backend behind a -E flag.
//
// Exit codes: Ok (0) if any match found, Fail (1) otherwise, Usage (2) on
// argument errors. Matches `grep` shell-script convention.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	ignoreCase, invert, withNumber := false, false, false
	var pos []string
	for _, a := range args {
		switch a {
		case "-i":
			ignoreCase = true
		case "-v":
			invert = true
		case "-n":
			withNumber = true
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) < 2 {
		fmt.Fprintln(env.Stderr, "grep: usage: grep [-i] [-v] [-n] PATTERN FILE...")
		return exit.Usage
	}
	pattern, files := pos[0], pos[1:]
	if ignoreCase {
		pattern = strings.ToLower(pattern)
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
			hay := line
			if ignoreCase {
				hay = strings.ToLower(hay)
			}
			match := strings.Contains(hay, pattern)
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
