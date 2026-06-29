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

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Source returns the list of "KEY=VALUE" strings env iterates over.
// Defaults to os.Environ; swappable from tests so the browser host (which
// has an empty real env) and the test suite can both drive deterministic
// output.
var Source = os.Environ

// Run prints the current environment, one KEY=VALUE per line, sorted by
// key. v0 does NOT implement assignment / -u / -i / COMMAND -- those grow
// in alongside subprocess support.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) > 0 {
		fmt.Fprintln(env.Stderr, "env: arguments are not supported (v0)")
		return exit.Usage
	}
	lines := append([]string(nil), Source()...)
	sort.Strings(lines)
	for _, l := range lines {
		fmt.Fprintln(env.Stdout, l)
	}
	return exit.Ok
}
