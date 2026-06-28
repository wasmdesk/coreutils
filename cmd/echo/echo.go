// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package echo

import (
	"fmt"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints env.Args[1:] joined by " " plus a trailing newline.
// Args[0] is the tool name (matching os.Args convention); the empty-args
// case produces a bare newline, which matches GNU `echo`.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	fmt.Fprintln(env.Stdout, strings.Join(args, " "))
	return exit.Ok
}
