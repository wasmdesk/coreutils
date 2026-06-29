// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package unexpand

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run converts runs of TABSTOP leading spaces (default 8) into a single tab.
// v0 only touches the LEADING whitespace of each line (matching the default
// GNU behaviour without -a).
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
				fmt.Fprintln(env.Stderr, "unexpand: option requires an argument: -t")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 1 {
				fmt.Fprintf(env.Stderr, "unexpand: invalid tabstop: %q\n", args[i])
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
		fmt.Fprintln(env.Stdout, packLeading(l, tabstop))
	}
	return exit.Ok
}

// packLeading collapses runs of TABSTOP leading spaces into a tab, leaving
// the rest of the line untouched.
func packLeading(s string, tabstop int) string {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	if i == 0 {
		return s
	}
	tabs := i / tabstop
	spaces := i % tabstop
	return strings.Repeat("\t", tabs) + strings.Repeat(" ", spaces) + s[i:]
}

func readAll(env *fsx.Env, paths []string) (string, int) {
	if len(paths) == 0 {
		if env.Stdin == nil {
			return "", exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "unexpand: stdin: %s\n", err)
			return "", exit.Fail
		}
		return string(b), exit.Ok
	}
	var sb strings.Builder
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "unexpand: %s: %s\n", p, prettyErr(err))
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
