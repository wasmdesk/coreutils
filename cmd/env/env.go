// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package envcmd is the import name for cmd/env (the package can't be
// called env without colliding with stdlib `env` usage in tests).
package envcmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Source returns the list of "KEY=VALUE" strings env starts from.
// Defaults to os.Environ; swappable from tests so the browser host (which
// has an empty real env) and the test suite can both drive deterministic
// output.
var Source = os.Environ

// Run is POSIX env: env [-i] [name=value]... [utility [argument...]].
// -i (or a lone -) starts from an empty environment; -u NAME removes a
// variable (GNU). With a utility, it runs it with that environment
// (looked up on the PATH it sets, if any) through env.Exec and exits with
// its status, 127 if it is not found, 126 if it cannot run. Without one,
// it prints the environment, one name=value per line, sorted by name.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	vars := Source()
	ignore := false
	var unset []string
	i := 0
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			i++
			break
		}
		if a == "-" || a == "-i" {
			ignore = true
			continue
		}
		if a == "-u" || (strings.HasPrefix(a, "-u") && len(a) > 2) {
			name := a[2:]
			if name == "" {
				i++
				if i >= len(args) {
					fmt.Fprintln(env.Stderr, "env: option requires an argument -- 'u'")
					return exit.Usage
				}
				name = args[i]
			}
			unset = append(unset, name)
			continue
		}
		if len(a) > 1 && a[0] == '-' {
			fmt.Fprintf(env.Stderr, "env: invalid option -- '%c'\n", a[1])
			return exit.Usage
		}
		break
	}
	if ignore {
		vars = nil
	}
	vars = without(vars, unset)
	// name=value operands, until the utility.
	for ; i < len(args) && strings.Contains(args[i], "="); i++ {
		k, _, _ := strings.Cut(args[i], "=")
		vars = append(without(vars, []string{k}), args[i])
	}
	if i == len(args) {
		lines := append([]string(nil), vars...)
		sort.Strings(lines)
		for _, l := range lines {
			fmt.Fprintln(env.Stdout, l)
		}
		return exit.Ok
	}
	if vars == nil {
		vars = []string{} // an empty environment, not the caller's
	}
	return env.RunStatus("env", fsx.Command{
		Name: args[i], Args: args[i:], Env: vars, Dir: env.Cwd,
		Stdin: env.Stdin, Stdout: env.Stdout, Stderr: env.Stderr,
	})
}

// without drops the variables named in names.
func without(vars, names []string) []string {
	if len(names) == 0 {
		return vars
	}
	out := vars[:0:0]
	for _, kv := range vars {
		k, _, _ := strings.Cut(kv, "=")
		drop := false
		for _, n := range names {
			drop = drop || k == n
		}
		if !drop {
			out = append(out, kv)
		}
	}
	return out
}
