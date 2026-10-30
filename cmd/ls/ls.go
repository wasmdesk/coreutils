// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ls

import (
	"errors"
	"fmt"
	"sort"

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
	// POSIX ls: an operand that is not a directory is listed by its name
	// (as given); those come first, then each directory's entries, under
	// a "dir:" header when there is more than one operand.
	rc := exit.Ok
	var files []fsx.FileInfo
	var dirs []string
	for _, p := range paths {
		st, err := env.FS.Stat(fsx.Resolve(env.Cwd, p))
		if err != nil {
			fmt.Fprintf(env.Stderr, "ls: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if st.IsDir {
			dirs = append(dirs, p)
			continue
		}
		st.Name = p
		files = append(files, st)
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	sort.Strings(dirs)
	show := func(e fsx.FileInfo) {
		switch {
		case long:
			kind := "-"
			if e.IsDir {
				kind = "d"
			}
			fmt.Fprintf(env.Stdout, "%s %8d %s\n", kind, e.Size, e.Name)
		case e.IsDir:
			fmt.Fprintf(env.Stdout, "%s/\n", e.Name)
		default:
			fmt.Fprintln(env.Stdout, e.Name)
		}
	}
	for _, f := range files {
		show(f)
	}
	for i, p := range dirs {
		entries, err := env.FS.ReadDir(fsx.Resolve(env.Cwd, p))
		if err != nil {
			fmt.Fprintf(env.Stderr, "ls: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		if len(paths) > 1 {
			if i > 0 || len(files) > 0 {
				fmt.Fprintln(env.Stdout)
			}
			fmt.Fprintf(env.Stdout, "%s:\n", p)
		}
		for _, e := range entries {
			show(e)
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
