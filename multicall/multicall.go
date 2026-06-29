// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package multicall is the single-source dispatch table for every coreutils
// tool. Both the wasmbox terminal builtin layer and the busybox-style single
// binary route through Dispatch(name, env) -- keeping the registry in one
// place means adding a new tool is a one-line entry.
package multicall

import (
	"fmt"

	"github.com/wasmdesk/coreutils/cmd/base32"
	"github.com/wasmdesk/coreutils/cmd/base64"
	"github.com/wasmdesk/coreutils/cmd/basename"
	"github.com/wasmdesk/coreutils/cmd/cat"
	"github.com/wasmdesk/coreutils/cmd/cp"
	"github.com/wasmdesk/coreutils/cmd/cut"
	"github.com/wasmdesk/coreutils/cmd/date"
	"github.com/wasmdesk/coreutils/cmd/dirname"
	"github.com/wasmdesk/coreutils/cmd/echo"
	envcmd "github.com/wasmdesk/coreutils/cmd/env"
	"github.com/wasmdesk/coreutils/cmd/expand"
	"github.com/wasmdesk/coreutils/cmd/expr"
	falsecmd "github.com/wasmdesk/coreutils/cmd/false"
	"github.com/wasmdesk/coreutils/cmd/find"
	"github.com/wasmdesk/coreutils/cmd/fold"
	"github.com/wasmdesk/coreutils/cmd/grep"
	"github.com/wasmdesk/coreutils/cmd/head"
	"github.com/wasmdesk/coreutils/cmd/ls"
	"github.com/wasmdesk/coreutils/cmd/md5sum"
	"github.com/wasmdesk/coreutils/cmd/mkdir"
	"github.com/wasmdesk/coreutils/cmd/mv"
	"github.com/wasmdesk/coreutils/cmd/nl"
	"github.com/wasmdesk/coreutils/cmd/paste"
	"github.com/wasmdesk/coreutils/cmd/printf"
	"github.com/wasmdesk/coreutils/cmd/pwd"
	"github.com/wasmdesk/coreutils/cmd/rev"
	"github.com/wasmdesk/coreutils/cmd/rm"
	"github.com/wasmdesk/coreutils/cmd/rmdir"
	"github.com/wasmdesk/coreutils/cmd/seq"
	"github.com/wasmdesk/coreutils/cmd/sha1sum"
	"github.com/wasmdesk/coreutils/cmd/sha256sum"
	"github.com/wasmdesk/coreutils/cmd/sleep"
	"github.com/wasmdesk/coreutils/cmd/sort"
	"github.com/wasmdesk/coreutils/cmd/tac"
	"github.com/wasmdesk/coreutils/cmd/tail"
	"github.com/wasmdesk/coreutils/cmd/touch"
	"github.com/wasmdesk/coreutils/cmd/tr"
	truecmd "github.com/wasmdesk/coreutils/cmd/true"
	"github.com/wasmdesk/coreutils/cmd/unexpand"
	"github.com/wasmdesk/coreutils/cmd/uniq"
	"github.com/wasmdesk/coreutils/cmd/wc"
	"github.com/wasmdesk/coreutils/cmd/yes"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// runFunc is the per-tool entry signature -- mirrors what each cmd/<tool>
// package exports as Run.
type runFunc func(env *fsx.Env) int

// registry is the name -> Run table. Built once at package init so Dispatch
// is a constant-time map lookup.
var registry = map[string]runFunc{
	// v0.2 (15 tools)
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
	// v0.3 (+27 tools): text processing
	"sort":     sort.Run,
	"uniq":     uniq.Run,
	"cut":      cut.Run,
	"tr":       tr.Run,
	"paste":    paste.Run,
	"nl":       nl.Run,
	"tac":      tac.Run,
	"rev":      rev.Run,
	"fold":     fold.Run,
	"expand":   expand.Run,
	"unexpand": unexpand.Run,
	"printf":   printf.Run,
	// v0.3: utility
	"basename": basename.Run,
	"dirname":  dirname.Run,
	"date":     date.Run,
	"seq":      seq.Run,
	"sleep":    sleep.Run,
	"true":     truecmd.Run,
	"false":    falsecmd.Run,
	"yes":      yes.Run,
	"env":      envcmd.Run,
	"expr":     expr.Run,
	// v0.3: crypto / encoding
	"md5sum":    md5sum.Run,
	"sha1sum":   sha1sum.Run,
	"sha256sum": sha256sum.Run,
	"base64":    base64.Run,
	"base32":    base32.Run,
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
