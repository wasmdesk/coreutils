// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ls

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run lists the entries of one or more paths. Flags: -l (long: type + size +
// name) and -a (no-op for now -- our FS has no hidden-dot convention, but the
// flag is accepted so terminal scripts that pass it don't break). Multiple
// path args are printed with a "path:" header. Missing args mean cwd.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	long, all := false, false
	var paths []string
	for _, a := range args {
		switch a {
		case "-l":
			long = true
		case "-a":
			all = true
		case "-la", "-al":
			long, all = true, true
		default:
			paths = append(paths, a)
		}
	}
	_ = all // accepted but no hidden-dot convention in fsx
	if len(paths) == 0 {
		paths = []string{env.Cwd}
	}
	rc := exit.Ok
	for i, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		entries, err := env.FS.ReadDir(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "ls: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if len(paths) > 1 {
			if i > 0 {
				fmt.Fprintln(env.Stdout)
			}
			fmt.Fprintf(env.Stdout, "%s:\n", p)
		}
		for _, e := range entries {
			if long {
				kind := "-"
				if e.IsDir {
					kind = "d"
				}
				fmt.Fprintf(env.Stdout, "%s %8d %s\n", kind, e.Size, e.Name)
				continue
			}
			if e.IsDir {
				fmt.Fprintf(env.Stdout, "%s/\n", e.Name)
				continue
			}
			fmt.Fprintln(env.Stdout, e.Name)
		}
	}
	return rc
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
