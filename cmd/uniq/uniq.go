// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package uniq

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run collapses adjacent duplicate lines in the supplied file (or stdin).
// Flags: -c prefix count, -d emit only repeated groups, -u emit only
// singletons. -c is incompatible with -d/-u in GNU; we accept any
// combination and let the simpler one win for v0.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	countOpt, onlyDup, onlyUniq := false, false, false
	var path string
	for _, a := range args {
		switch a {
		case "-c":
			countOpt = true
		case "-d":
			onlyDup = true
		case "-u":
			onlyUniq = true
		default:
			if path != "" {
				fmt.Fprintln(env.Stderr, "uniq: extra operand: "+a)
				return exit.Usage
			}
			path = a
		}
	}
	var data string
	if path == "" {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "uniq: stdin: %s\n", err)
			return exit.Fail
		}
		data = string(b)
	} else {
		abs := fsx.Resolve(env.Cwd, path)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "uniq: %s: %s\n", path, prettyErr(err))
			return exit.Fail
		}
		data = string(b)
	}
	lines := splitLines(data)
	emit := func(line string, n int) {
		if onlyDup && n < 2 {
			return
		}
		if onlyUniq && n != 1 {
			return
		}
		if countOpt {
			fmt.Fprintf(env.Stdout, "%7d %s\n", n, line)
			return
		}
		fmt.Fprintln(env.Stdout, line)
	}
	if len(lines) == 0 {
		return exit.Ok
	}
	curr := lines[0]
	count := 1
	for _, l := range lines[1:] {
		if l == curr {
			count++
			continue
		}
		emit(curr, count)
		curr = l
		count = 1
	}
	emit(curr, count)
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
