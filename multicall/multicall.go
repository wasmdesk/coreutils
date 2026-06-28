// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package multicall is the single-source dispatch table for every coreutils
// tool. Both the wasmbox terminal builtin layer and the busybox-style single
// binary route through Dispatch(name, env) -- keeping the registry in one
// place means adding a new tool (Phase 1: ~85 to go) is a one-line entry.
package multicall

import (
	"fmt"

	"github.com/wasmdesk/coreutils/cmd/cat"
	"github.com/wasmdesk/coreutils/cmd/cp"
	"github.com/wasmdesk/coreutils/cmd/echo"
	"github.com/wasmdesk/coreutils/cmd/find"
	"github.com/wasmdesk/coreutils/cmd/grep"
	"github.com/wasmdesk/coreutils/cmd/head"
	"github.com/wasmdesk/coreutils/cmd/ls"
	"github.com/wasmdesk/coreutils/cmd/mkdir"
	"github.com/wasmdesk/coreutils/cmd/mv"
	"github.com/wasmdesk/coreutils/cmd/pwd"
	"github.com/wasmdesk/coreutils/cmd/rm"
	"github.com/wasmdesk/coreutils/cmd/rmdir"
	"github.com/wasmdesk/coreutils/cmd/tail"
	"github.com/wasmdesk/coreutils/cmd/touch"
	"github.com/wasmdesk/coreutils/cmd/wc"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// runFunc is the per-tool entry signature -- mirrors what each cmd/<tool>
// package exports as Run.
type runFunc func(env *fsx.Env) int

// registry is the name -> Run table. Built once at package init so Dispatch
// is a constant-time map lookup.
var registry = map[string]runFunc{
	"pwd":   pwd.Run,
	"echo":  echo.Run,
	"cat":   cat.Run,
	"ls":    ls.Run,
	"mkdir": mkdir.Run,
	"rmdir": rmdir.Run,
	"rm":    rm.Run,
	"cp":    cp.Run,
	"mv":    mv.Run,
	"touch": touch.Run,
	"head":  head.Run,
	"tail":  tail.Run,
	"wc":    wc.Run,
	"grep":  grep.Run,
	"find":  find.Run,
}

// Dispatch routes (name, env) to the matching tool's Run. Unknown names
// return Fail and emit a "command not found" line to env.Stderr -- matching
// the shape a real shell would emit when PATH lookup fails.
func Dispatch(name string, env *fsx.Env) int {
	fn, ok := registry[name]
	if !ok {
		fmt.Fprintf(env.Stderr, "%s: command not found\n", name)
		return exit.Fail
	}
	return fn(env)
}

// Has reports whether name is a registered tool. The wasmbox shell uses this
// to decide whether to route a typed command through Dispatch or fall back
// to its "command not found" message.
func Has(name string) bool {
	_, ok := registry[name]
	return ok
}

// Names returns the alphabetised tool list. Handy for help-text generators
// and tests; the wasmbox terminal builds its `help` line from this.
func Names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	// Simple insertion sort to avoid pulling in the sort package just for
	// this; the tool list is tiny and grows linearly per release.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
