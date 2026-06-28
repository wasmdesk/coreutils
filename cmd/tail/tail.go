// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tail

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints the last N lines (default 10) of each file. -n N selects the
// count. With multiple files, a "==> name <==" header preceeds each block.
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
				fmt.Fprintln(env.Stderr, "tail: option requires an argument: -n")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 0 {
				fmt.Fprintf(env.Stderr, "tail: invalid line count: %q\n", args[i])
				return exit.Usage
			}
			n = v
			continue
		}
		paths = append(paths, a)
	}
	if len(paths) == 0 {
		fmt.Fprintln(env.Stderr, "tail: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for i, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		data, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "tail: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Fprintln(env.Stdout)
			}
			fmt.Fprintf(env.Stdout, "==> %s <==\n", p)
		}
		printLastN(env.Stdout, data, n)
	}
	return rc
}

// printLastN slices the trailing n lines off data and writes them. If the
// file has fewer than n lines, the whole body is printed verbatim.
func printLastN(w writer, data []byte, n int) {
	if n == 0 {
		return
	}
	s := string(data)
	// Split into lines without losing the trailing-newline distinction.
	// We materialise lines by repeatedly slicing on '\n'.
	lines := []string{}
	rest := s
	for rest != "" {
		i := strings.IndexByte(rest, '\n')
		if i < 0 {
			lines = append(lines, rest)
			rest = ""
			break
		}
		lines = append(lines, rest[:i]+"\n")
		rest = rest[i+1:]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, ln := range lines {
		fmt.Fprint(w, ln)
	}
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
