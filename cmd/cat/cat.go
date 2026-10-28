// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cat

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run concatenates the listed files to env.Stdout. Supports `-n` (number
// every output line, GNU style: a right-justified 6-wide line number + tab).
// With no operand, or for the operand "-", it copies standard input
// (POSIX cat).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	number := false
	paths := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "-n":
			number = true
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		paths = []string{fsx.StdinOperand}
	}
	rc := exit.Ok
	line := 1
	for _, p := range paths {
		if p == fsx.StdinOperand && !number {
			// Stream: `producer | cat` must not wait for the producer to end.
			if env.Stdin != nil {
				if _, err := io.Copy(env.Stdout, env.Stdin); err != nil {
					fmt.Fprintf(env.Stderr, "cat: -: %s\n", err)
					rc = exit.Fail
				}
			}
			continue
		}
		data, err := env.ReadOperand(p)
		if err != nil {
			fmt.Fprintf(env.Stderr, "cat: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if !number {
			env.Stdout.Write(data)
			continue
		}
		// Numbered output: split on '\n', print each non-trailing line with a
		// counter prefix. The trailing-newline-on-last-line case is preserved.
		text := string(data)
		for text != "" {
			i := strings.IndexByte(text, '\n')
			if i < 0 {
				fmt.Fprintf(env.Stdout, "%6d\t%s", line, text)
				line++
				break
			}
			fmt.Fprintf(env.Stdout, "%6d\t%s\n", line, text[:i])
			line++
			text = text[i+1:]
		}
	}
	return rc
}

// prettyErr maps an fsx sentinel onto the short suffix shells expect.
func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
