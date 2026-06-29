// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package paste

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run merges corresponding lines from each named file side by side,
// separated by DELIM (default TAB). When sources are uneven, missing
// columns become empty strings. v0 does NOT read stdin (the wasmbox
// shell pipeline has no pipes) -- at least one path is required.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	delim := "\t"
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-d" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "paste: option requires an argument: -d")
				return exit.Usage
			}
			delim = args[i]
			continue
		}
		paths = append(paths, a)
	}
	if len(paths) == 0 {
		fmt.Fprintln(env.Stderr, "paste: missing operand")
		return exit.Usage
	}
	cols := make([][]string, len(paths))
	maxLen := 0
	for i, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "paste: %s: %s\n", p, prettyErr(err))
			return exit.Fail
		}
		cols[i] = splitLines(string(b))
		if len(cols[i]) > maxLen {
			maxLen = len(cols[i])
		}
	}
	for row := 0; row < maxLen; row++ {
		parts := make([]string, len(cols))
		for i, c := range cols {
			if row < len(c) {
				parts[i] = c[row]
			}
		}
		fmt.Fprintln(env.Stdout, strings.Join(parts, delim))
	}
	return exit.Ok
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

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
