// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package head

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints the first N lines (default 10) of each named file. -n N selects
// the line count. With multiple files, a "==> name <==" header preceeds
// each block. Missing operand -> Usage. Negative N is rejected.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	n := 10
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-n" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "head: option requires an argument: -n")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 0 {
				fmt.Fprintf(env.Stderr, "head: invalid line count: %q\n", args[i])
				return exit.Usage
			}
			n = v
			continue
		}
		paths = append(paths, a)
	}
	if len(paths) == 0 {
		fmt.Fprintln(env.Stderr, "head: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for i, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		data, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "head: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Fprintln(env.Stdout)
			}
			fmt.Fprintf(env.Stdout, "==> %s <==\n", p)
		}
		printFirstN(env.Stdout, data, n)
	}
	return rc
}

// printFirstN writes up to n newline-delimited lines from data, preserving
// the trailing newline (or its absence) on the last printed line.
func printFirstN(w writer, data []byte, n int) {
	if n == 0 {
		return
	}
	count := 0
	text := string(data)
	for text != "" && count < n {
		i := strings.IndexByte(text, '\n')
		if i < 0 {
			fmt.Fprint(w, text)
			return
		}
		fmt.Fprintln(w, text[:i])
		text = text[i+1:]
		count++
	}
}

// writer is the minimal io.Writer-shaped interface printFirstN needs. Kept
// local so the test file does not import io for a single type.
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
