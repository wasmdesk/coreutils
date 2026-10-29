// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package uname implements the uname coreutil.
package uname

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/wasmdesk/coreutils/cmd/hostname"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// GOOS and GOARCH are what the fields are derived from; swappable from
// tests.
var GOOS, GOARCH = runtime.GOOS, runtime.GOARCH

// fields returns the kernel name, release, version, machine and
// operating system. A WASI program has no kernel to ask: under wasip1 it
// reports the interface it runs on.
func fields() (sys, release, version, machine, osName string) {
	switch GOOS {
	case "wasip1":
		return "WASI", "preview1", "wasip1", "wasm32", "WASI"
	case "js":
		return "Browser", "js", "js", "wasm32", "Browser"
	}
	sys = strings.ToUpper(GOOS[:1]) + GOOS[1:]
	if GOOS == "darwin" {
		sys = "Darwin"
	}
	machine = map[string]string{"amd64": "x86_64", "arm64": "arm64", "386": "i686"}[GOARCH]
	if machine == "" {
		machine = GOARCH
	}
	return sys, "unknown", "unknown", machine, sys
}

// Run prints system information, POSIX uname: -s (default) -n -r -v -m,
// -a for all, plus GNU's -o; -p and -i print "unknown".
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	want := map[byte]bool{}
	for _, a := range args {
		if a == "--all" {
			a = "-a"
		}
		if len(a) < 2 || a[0] != '-' {
			fmt.Fprintf(env.Stderr, "uname: extra operand %q\n", a)
			return exit.Usage
		}
		for _, c := range []byte(a[1:]) {
			if !strings.ContainsRune("asnrvmopi", rune(c)) {
				fmt.Fprintf(env.Stderr, "uname: invalid option -- '%c'\n", c)
				return exit.Usage
			}
			want[c] = true
		}
	}
	if len(want) == 0 {
		want['s'] = true
	}
	sys, release, version, machine, osName := fields()
	all := want['a']
	var out []string
	add := func(c byte, v string) {
		if all || want[c] {
			out = append(out, v)
		}
	}
	add('s', sys)
	add('n', hostname.Name())
	add('r', release)
	add('v', version)
	add('m', machine)
	if !all {
		add('p', "unknown")
		add('i', "unknown")
	}
	add('o', osName)
	fmt.Fprintln(env.Stdout, strings.Join(out, " "))
	return exit.Ok
}
