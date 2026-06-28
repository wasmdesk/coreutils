// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Command multicall is the busybox-style single binary: argv[0] selects the
// tool when the binary is invoked through a symlink, otherwise argv[1] does.
// Useful for `task run -- <tool> <args>` and for shipping a one-binary
// distribution.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wasmdesk/coreutils/multicall"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func main() {
	args := os.Args
	name := filepath.Base(args[0])
	// If the binary was invoked under its own "multicall" name, the first
	// positional arg is the tool name.
	if name == "multicall" || name == "main" || name == "main.exe" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "multicall: usage: multicall TOOL [args...]")
			fmt.Fprintf(os.Stderr, "  tools: %v\n", multicall.Names())
			os.Exit(2)
		}
		name = args[1]
		args = args[1:]
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	os.Exit(multicall.Dispatch(name, &fsx.Env{
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Cwd:    cwd,
		FS:     fsx.NewOSFS(),
	}))
}
