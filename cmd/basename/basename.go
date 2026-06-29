// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package basename

import (
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints the trailing path component of its single PATH argument. An
// optional second SUFFIX argument is stripped from the result if present.
// Mirrors GNU `basename NAME [SUFFIX]` -- no flags, no multi-name -a mode.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "basename: missing operand")
		return exit.Usage
	}
	if len(args) > 2 {
		fmt.Fprintln(env.Stderr, "basename: too many operands")
		return exit.Usage
	}
	name := trimTrailingSlashes(args[0])
	// Strip directory prefix. The literal "/" survives trim-trailing as "/"
	// and stays as-is (GNU prints "/" for basename "/").
	if name != "/" {
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
	}
	if len(args) == 2 {
		suf := args[1]
		if suf != "" && name != suf && strings.HasSuffix(name, suf) {
			name = name[:len(name)-len(suf)]
		}
	}
	fmt.Fprintln(env.Stdout, name)
	return exit.Ok
}

// trimTrailingSlashes drops one-or-more trailing "/" characters but never
// collapses the literal root "/" to an empty string -- matching GNU basename.
func trimTrailingSlashes(p string) string {
	if p == "" {
		return p
	}
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}
