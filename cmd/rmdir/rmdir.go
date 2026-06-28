// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package rmdir

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run removes empty directories. No flags in v0 (matching GNU's bare
// rmdir behaviour); non-empty dirs and files report errors and continue
// onto the next operand.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "rmdir: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for _, p := range args {
		abs := fsx.Resolve(env.Cwd, p)
		info, sErr := env.FS.Stat(abs)
		if sErr != nil {
			fmt.Fprintf(env.Stderr, "rmdir: %s: %s\n", p, prettyErr(sErr))
			rc = exit.Fail
			continue
		}
		if !info.IsDir {
			fmt.Fprintf(env.Stderr, "rmdir: %s: not a directory\n", p)
			rc = exit.Fail
			continue
		}
		if err := env.FS.Remove(abs); err != nil {
			fmt.Fprintf(env.Stderr, "rmdir: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrNotEmpty):
		return "directory not empty"
	case errors.Is(err, fsx.ErrInvalid):
		return "invalid argument"
	}
	return err.Error()
}
