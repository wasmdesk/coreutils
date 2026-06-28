// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package touch

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run creates each named file if missing (empty body). Already-existing
// paths are a no-op -- the FS contract carries no mtime, so we cannot honour
// the "update modification time" half of GNU touch and we say so in the
// README. Missing operand -> Usage.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "touch: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	for _, p := range args {
		abs := fsx.Resolve(env.Cwd, p)
		if _, err := env.FS.Stat(abs); err == nil {
			continue
		}
		if err := env.FS.WriteFile(abs, nil); err != nil {
			fmt.Fprintf(env.Stderr, "touch: %s: %s\n", p, prettyErr(err))
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
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
