// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package which implements which: where a command is found on $PATH.
package which

import (
	"fmt"
	"os"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// LookupEnv reads $PATH; swappable from tests and the browser host.
var LookupEnv = os.LookupEnv

// Candidates are the files a name may be, per PATH directory: the name,
// then name.wasm (a WebAssembly program's file name, as in pkgx bottles
// and the go-wasmcontainers shell's PATH lookup).
func Candidates(dir, name string) []string {
	p := fsx.Join(dir, name)
	if strings.HasSuffix(name, ".wasm") {
		return []string{p}
	}
	return []string{p, p + ".wasm"}
}

// Find returns the paths name resolves to on pathList (all of them with
// all, else the first). A name with a slash is checked as given.
func Find(fs fsx.FS, cwd, pathList, name string, all bool) []string {
	isFile := func(p string) bool {
		st, err := fs.Stat(p)
		return err == nil && !st.IsDir
	}
	if strings.Contains(name, "/") {
		for _, c := range Candidates(fsx.Parent(fsx.Resolve(cwd, name)), fsx.Basename(name)) {
			if isFile(c) {
				return []string{c}
			}
		}
		return nil
	}
	var out []string
	for _, d := range strings.Split(pathList, ":") {
		if d == "" {
			d = "."
		}
		for _, c := range Candidates(fsx.Resolve(cwd, d), name) {
			if isFile(c) {
				out = append(out, c)
				if !all {
					return out
				}
				break
			}
		}
	}
	return out
}

// Run is which [-a] NAME...: exit 1 if any name is not found.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	all := false
	var names []string
	for _, a := range args {
		if a == "-a" {
			all = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			fmt.Fprintf(env.Stderr, "which: invalid option %q\n", a)
			return exit.Usage
		}
		names = append(names, a)
	}
	if len(names) == 0 {
		fmt.Fprintln(env.Stderr, "which: missing operand")
		return exit.Usage
	}
	pathList, _ := LookupEnv("PATH")
	rc := exit.Ok
	for _, n := range names {
		found := Find(env.FS, env.Cwd, pathList, n, all)
		if len(found) == 0 {
			rc = exit.Fail
		}
		for _, p := range found {
			fmt.Fprintln(env.Stdout, p)
		}
	}
	return rc
}
