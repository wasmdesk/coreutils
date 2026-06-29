// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fold

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run wraps each input line at WIDTH columns (default 80). Lines shorter
// than WIDTH pass through; long lines are split into chunks. v0 does not
// implement -s (word-aware) wrapping; that lands when we need it.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	width := 80
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-w" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "fold: option requires an argument: -w")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 1 {
				fmt.Fprintf(env.Stderr, "fold: invalid width: %q\n", args[i])
				return exit.Usage
			}
			width = v
			continue
		}
		paths = append(paths, a)
	}
	var data string
	if len(paths) == 0 {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "fold: stdin: %s\n", err)
			return exit.Fail
		}
		data = string(b)
	} else {
		var sb strings.Builder
		for _, p := range paths {
			abs := fsx.Resolve(env.Cwd, p)
			b, err := env.FS.ReadFile(abs)
			if err != nil {
				fmt.Fprintf(env.Stderr, "fold: %s: %s\n", p, prettyErr(err))
				return exit.Fail
			}
			sb.Write(b)
		}
		data = sb.String()
	}
	for _, l := range splitLines(data) {
		for len(l) > width {
			fmt.Fprintln(env.Stdout, l[:width])
			l = l[width:]
		}
		fmt.Fprintln(env.Stdout, l)
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
