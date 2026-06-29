// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package md5sum

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints "<hash>  <file>" for each operand. With no operands, hashes
// stdin and prints "<hash>  -" (matching GNU md5sum).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "md5sum: stdin: %s\n", err)
			return exit.Fail
		}
		sum := md5.Sum(b)
		fmt.Fprintf(env.Stdout, "%s  -\n", hex.EncodeToString(sum[:]))
		return exit.Ok
	}
	rc := exit.Ok
	for _, p := range args {
		abs := fsx.Resolve(env.Cwd, p)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "md5sum: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		sum := md5.Sum(b)
		fmt.Fprintf(env.Stdout, "%s  %s\n", hex.EncodeToString(sum[:]), p)
	}
	return rc
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
