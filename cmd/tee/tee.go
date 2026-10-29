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

// Run copies standard input to standard output and to each FILE as it
// arrives (POSIX tee [-a] [-i] [FILE...]; -i is accepted, there are no
// signals to ignore). The FILEs are opened first — emptied, or with -a
// appended to — so `tee log` can be watched while it runs. On a
// filesystem that only writes whole files (no fsx.Opener), the FILEs are
// written when the input ends.
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
	rc := exit.Ok
	fail := func(name string, err error) {
		fmt.Fprintf(env.Stderr, "tee: %s: %v\n", name, err)
		rc = exit.Fail
	}
	type output struct {
		name string
		w    io.WriteCloser
	}
	var outs []output
	for _, f := range files {
		w, err := fsx.Create(env.FS, fsx.Resolve(env.Cwd, f), appendMode)
		if err != nil {
			fail(f, err) // the other FILEs still get the input
			continue
		}
		outs = append(outs, output{f, w})
	}
	stdoutOK := true
	if env.Stdin != nil {
		buf := make([]byte, 32<<10)
		for {
			n, err := env.Stdin.Read(buf)
			if n > 0 {
				if _, werr := env.Stdout.Write(buf[:n]); werr != nil && stdoutOK {
					fail("standard output", werr)
					stdoutOK = false
				}
				for i := 0; i < len(outs); {
					if _, werr := outs[i].w.Write(buf[:n]); werr != nil {
						fail(outs[i].name, werr) // reported once, then dropped
						outs[i].w.Close()
						outs = append(outs[:i], outs[i+1:]...)
						continue
					}
					i++
				}
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					fail("standard input", err)
				}
				break
			}
		}
	}
	for _, o := range outs {
		if err := o.w.Close(); err != nil {
			fail(o.name, err)
		}
	}
	return rc
}
