// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command printf is the native CLI wrapper around cmd/printf.Run.
package main

import (
	"os"

	"github.com/wasmdesk/coreutils/cmd/printf"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	os.Exit(printf.Run(&fsx.Env{
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Cwd:    cwd,
		FS:     fsx.NewOSFS(),
	}))
}
