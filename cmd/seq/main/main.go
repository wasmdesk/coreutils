// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command seq is the native CLI wrapper around cmd/seq.Run.
package main

import (
	"os"

	"github.com/wasmdesk/coreutils/cmd/seq"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	os.Exit(seq.Run(&fsx.Env{
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Cwd:    cwd,
		FS:     fsx.NewOSFS(),
	}))
}
