// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package expand

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run converts tab characters in each input line to enough spaces to land on
// the next multiple of TABSTOP (default 8). Flag: -t N.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	tabstop := 8
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-t" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "expand: option requires an argument: -t")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 1 {
				fmt.Fprintf(env.Stderr, "expand: invalid tabstop: %q\n", args[i])
				return exit.Usage
			}
			tabstop = v
			continue
		}
		paths = append(paths, a)
	}
	data, rc := readAll(env, paths)
	if rc != exit.Ok {
		return rc
	}
	for _, l := range splitLines(data) {
		fmt.Fprintln(env.Stdout, expandTabs(l, tabstop))
	}
	return exit.Ok
}

// expandTabs replaces every '\t' in s with spaces up to the next multiple of
// tabstop.
func expandTabs(s string, tabstop int) string {
	var b strings.Builder
	col := 0
	for _, r := range s {
		if r == '\t' {
			n := tabstop - col%tabstop
			for i := 0; i < n; i++ {
				b.WriteByte(' ')
				col++
			}
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

func readAll(env *fsx.Env, paths []string) (string, int) {
	if len(paths) == 0 {
		if env.Stdin == nil {
			return "", exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "expand: stdin: %s\n", err)
			return "", exit.Fail
		}
		return string(b), exit.Ok
	}
	var sb strings.Builder
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "expand: %s: %s\n", p, prettyErr(err))
			return "", exit.Fail
		}
		sb.Write(b)
	}
	return sb.String(), exit.Ok
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
