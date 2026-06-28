// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package rm

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run removes files. Flags: -r/-R (recursive, lets directories be removed
// too) and -f (force -- missing files are not errors and no usage hint when
// no operand is given). Composite flags (-rf, -fr) are accepted.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	recursive, force := false, false
	var paths []string
	for _, a := range args {
		switch a {
		case "-r", "-R":
			recursive = true
		case "-f":
			force = true
		case "-rf", "-fr", "-Rf", "-fR":
			recursive, force = true, true
		default:
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		if force {
			return exit.Ok
		}
		fmt.Fprintln(env.Stderr, "rm: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		info, sErr := env.FS.Stat(abs)
		if sErr != nil {
			if force {
				continue
			}
			fmt.Fprintf(env.Stderr, "rm: %s: %s\n", p, prettyErr(sErr))
			rc = exit.Fail
			continue
		}
		if info.IsDir && !recursive {
			fmt.Fprintf(env.Stderr, "rm: %s: is a directory\n", p)
			rc = exit.Fail
			continue
		}
		var err error
		if recursive {
			err = env.FS.RemoveAll(abs)
		} else {
			err = env.FS.Remove(abs)
		}
		if err != nil {
			if force {
				continue
			}
			fmt.Fprintf(env.Stderr, "rm: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	case errors.Is(err, fsx.ErrNotEmpty):
		return "directory not empty"
	case errors.Is(err, fsx.ErrInvalid):
		return "invalid argument"
	}
	return err.Error()
}
