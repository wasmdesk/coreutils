// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package falsecmd is the import name for cmd/false (Go would not let us call
// the package `false`, since that is a predeclared identifier). The tool
// name stays `false` in argv / the multicall registry.
package falsecmd

import (
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run always returns failure. Arguments are ignored (GNU semantics).
func Run(env *fsx.Env) int {
	_ = env
	return exit.Fail
}
