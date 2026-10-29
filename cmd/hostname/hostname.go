// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package hostname implements the hostname coreutil.
package hostname

import (
	"fmt"
	"os"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// LookupEnv and Hostname are the sources Name reads; swappable from tests
// and from the browser host.
var (
	LookupEnv = os.LookupEnv
	Hostname  = os.Hostname
)

// Name is the host's name: $HOSTNAME (what a container engine sets, and
// the only place a WASI program can learn it), else the OS's answer, else
// "localhost".
func Name() string {
	if h, ok := LookupEnv("HOSTNAME"); ok && h != "" {
		return h
	}
	if h, err := Hostname(); err == nil && h != "" {
		return h
	}
	return "localhost"
}

// Run prints the host name. Setting it (an operand) is refused: neither a
// WASI program nor the browser can change it.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) > 0 {
		fmt.Fprintln(env.Stderr, "hostname: setting the host name is not supported")
		return exit.Usage
	}
	fmt.Fprintln(env.Stdout, Name())
	return exit.Ok
}
