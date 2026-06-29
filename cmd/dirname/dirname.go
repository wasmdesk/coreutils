// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package dirname

import (
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints the parent directory portion of each PATH argument, one per
// line. Matches GNU `dirname` -- bare component (no slash) -> ".", "/" -> "/".
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "dirname: missing operand")
		return exit.Usage
	}
	for _, p := range args {
		fmt.Fprintln(env.Stdout, dirOf(p))
	}
	return exit.Ok
}

// dirOf returns the directory portion of p with GNU semantics: trailing
// slashes are stripped before splitting; a path without "/" yields ".";
// the root stays "/".
func dirOf(p string) string {
	if p == "" {
		return "."
	}
	// Strip trailing slashes, but keep at least one char so "/" survives.
	for len(p) > 1 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "."
	}
	if i == 0 {
		return "/"
	}
	d := p[:i]
	// Collapse any trailing slashes that appear once we strip the basename
	// (e.g. "a//b" -> "a").
	for len(d) > 1 && d[len(d)-1] == '/' {
		d = d[:len(d)-1]
	}
	return d
}
