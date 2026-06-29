// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sort

import (
	"errors"
	"fmt"
	"io"
	stdsort "sort"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run reads the concatenated input lines from each named file (or env.Stdin
// when no files are given) and prints them in order. Flags: -r reverse,
// -n numeric, -u drop duplicates after sorting.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	reverse, numeric, unique := false, false, false
	var paths []string
	for _, a := range args {
		switch a {
		case "-r":
			reverse = true
		case "-n":
			numeric = true
		case "-u":
			unique = true
		default:
			paths = append(paths, a)
		}
	}
	lines, rc := collect(env, paths)
	if rc != exit.Ok {
		return rc
	}
	if numeric {
		stdsort.SliceStable(lines, func(i, j int) bool {
			return numLess(lines[i], lines[j])
		})
	} else {
		stdsort.Strings(lines)
	}
	if reverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}
	if unique {
		lines = dedup(lines)
	}
	for _, l := range lines {
		fmt.Fprintln(env.Stdout, l)
	}
	return exit.Ok
}

// collect reads every input source into a flat slice of newline-stripped
// lines. Each source contributes its content top-to-bottom; the final entry
// is dropped if the file ended on a newline.
func collect(env *fsx.Env, paths []string) ([]string, int) {
	var lines []string
	if len(paths) == 0 {
		if env.Stdin == nil {
			return nil, exit.Ok
		}
		data, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "sort: stdin: %s\n", err)
			return nil, exit.Fail
		}
		return splitLines(string(data)), exit.Ok
	}
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		data, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "sort: %s: %s\n", p, prettyErr(err))
			return nil, exit.Fail
		}
		lines = append(lines, splitLines(string(data))...)
	}
	return lines, exit.Ok
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// numLess returns true if a < b interpreted as leading integers. Lines that
// don't parse fall through to a lexicographic tie-breaker.
func numLess(a, b string) bool {
	ai, aerr := strconv.ParseFloat(strings.TrimSpace(a), 64)
	bi, berr := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if aerr == nil && berr == nil {
		return ai < bi
	}
	return a < b
}

// dedup removes adjacent duplicates from a sorted slice in place.
func dedup(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
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
