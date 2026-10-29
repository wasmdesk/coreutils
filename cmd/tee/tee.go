// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package tee implements the tee coreutil.
package tee

import (
	"errors"
	"fmt"
	"io"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run copies standard input to standard output as it arrives, and to each
// FILE (POSIX tee [-a] [-i] [FILE...]; -i is accepted, there are no
// signals to ignore). The fsx filesystem writes whole files, so the FILEs
// are written when the input ends — after the previous content, with -a.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	appendMode := false
	var files []string
	for i, a := range args {
		if a == "--" {
			files = append(files, args[i+1:]...)
			break
		}
		if len(a) > 1 && a[0] == '-' {
			for _, c := range a[1:] {
				switch c {
				case 'a':
					appendMode = true
				case 'i':
				default:
					fmt.Fprintf(env.Stderr, "tee: invalid option -- '%c'\n", c)
					return exit.Usage
				}
			}
			continue
		}
		files = append(files, a)
	}
	var data []byte
	rc := exit.Ok
	if env.Stdin != nil {
		buf := make([]byte, 32<<10)
		for {
			n, err := env.Stdin.Read(buf)
			if n > 0 {
				data = append(data, buf[:n]...)
				if _, werr := env.Stdout.Write(buf[:n]); werr != nil && rc == exit.Ok {
					fmt.Fprintf(env.Stderr, "tee: standard output: %v\n", werr)
					rc = exit.Fail
				}
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					fmt.Fprintf(env.Stderr, "tee: standard input: %v\n", err)
					rc = exit.Fail
				}
				break
			}
		}
	}
	for _, f := range files {
		abs := fsx.Resolve(env.Cwd, f)
		out := data
		if appendMode {
			old, err := env.FS.ReadFile(abs)
			if err == nil {
				out = append(append([]byte(nil), old...), data...)
			} else if !errors.Is(err, fsx.ErrNotFound) {
				fmt.Fprintf(env.Stderr, "tee: %s: %v\n", f, err)
				rc = exit.Fail
				continue
			}
		}
		if err := env.FS.WriteFile(abs, out); err != nil {
			fmt.Fprintf(env.Stderr, "tee: %s: %v\n", f, err)
			rc = exit.Fail
		}
	}
	return rc
}
