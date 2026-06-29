// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tac

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run reverses the line order of each named file (or stdin) and writes the
// result to env.Stdout. Multiple files are concatenated, then reversed as a
// single stream (matching GNU tac).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var data string
	if len(args) == 0 {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "tac: stdin: %s\n", err)
			return exit.Fail
		}
		data = string(b)
	} else {
		var sb strings.Builder
		for _, p := range args {
			abs := fsx.Resolve(env.Cwd, p)
			b, err := env.FS.ReadFile(abs)
			if err != nil {
				fmt.Fprintf(env.Stderr, "tac: %s: %s\n", p, prettyErr(err))
				return exit.Fail
			}
			sb.Write(b)
		}
		data = sb.String()
	}
	lines := splitLines(data)
	for i := len(lines) - 1; i >= 0; i-- {
		fmt.Fprintln(env.Stdout, lines[i])
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
