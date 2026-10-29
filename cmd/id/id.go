// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package id implements the id and whoami coreutils.
//
// A WASI container (and the browser) has no users: every program runs as
// root, uid 0, gid 0. These tools say so, in the formats scripts parse.
package id

import (
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run is POSIX id [-u|-g|-G] [-n] [-r] [USER]; USER may only be root.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var u, g, G, n bool
	for _, a := range args {
		if !strings.HasPrefix(a, "-") || a == "-" {
			if a != "root" && a != "0" {
				fmt.Fprintf(env.Stderr, "id: '%s': no such user\n", a)
				return exit.Fail
			}
			continue
		}
		for _, c := range a[1:] {
			switch c {
			case 'u':
				u = true
			case 'g':
				g = true
			case 'G':
				G = true
			case 'n':
				n = true
			case 'r':
			default:
				fmt.Fprintf(env.Stderr, "id: invalid option -- '%c'\n", c)
				return exit.Usage
			}
		}
	}
	if (u && g) || (u && G) || (g && G) {
		fmt.Fprintln(env.Stderr, "id: cannot print \"only\" of more than one choice")
		return exit.Usage
	}
	switch {
	case u || g || G:
		if n {
			fmt.Fprintln(env.Stdout, "root")
		} else {
			fmt.Fprintln(env.Stdout, "0")
		}
	case n:
		fmt.Fprintln(env.Stderr, "id: cannot print only names or real IDs in default format")
		return exit.Usage
	default:
		fmt.Fprintln(env.Stdout, "uid=0(root) gid=0(root) groups=0(root)")
	}
	return exit.Ok
}

// Whoami prints the effective user's name: always root.
func Whoami(env *fsx.Env) int {
	if len(env.Args) > 1 {
		fmt.Fprintf(env.Stderr, "whoami: extra operand '%s'\n", env.Args[1])
		return exit.Usage
	}
	fmt.Fprintln(env.Stdout, "root")
	return exit.Ok
}
