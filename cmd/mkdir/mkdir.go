// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mkdir

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run creates one or more directories. Supports -p (create parents as
// needed; never fail on an existing dir). Missing operand -> Usage.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	parents := false
	paths := paths(args, &parents)
	if len(paths) == 0 {
		fmt.Fprintln(env.Stderr, "mkdir: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		var err error
		if parents {
			err = env.FS.MkdirAll(abs)
		} else {
			err = env.FS.Mkdir(abs)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "mkdir: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

func paths(args []string, parents *bool) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "-p" {
			*parents = true
			continue
		}
		out = append(out, a)
	}
	return out
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrExists):
		return "file exists"
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
