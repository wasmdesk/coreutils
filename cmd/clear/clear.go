// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package clear implements clear: the ANSI sequence that homes the cursor,
// erases the screen and, unless -x, the scrollback (xterm's E3).
package clear

import (
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Sequences written by clear.
const (
	Screen     = "\x1b[H\x1b[2J"
	Scrollback = "\x1b[3J"
)

// Run writes the sequence; -x keeps the scrollback (ncurses clear -x).
func Run(env *fsx.Env) int {
	keep := false
	for _, a := range env.Args[min(1, len(env.Args)):] {
		if a != "-x" {
			fmt.Fprintf(env.Stderr, "clear: unknown option %q\n", a)
			return exit.Usage
		}
		keep = true
	}
	s := Screen
	if !keep {
		s += Scrollback
	}
	fmt.Fprint(env.Stdout, s)
	return exit.Ok
}
