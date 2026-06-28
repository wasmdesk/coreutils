// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package pwd

import (
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints env.Cwd + "\n" and returns Ok. There is no argument to parse --
// extra args are ignored to mirror what GNU pwd does when called from a
// dispatcher (it simply prints $PWD regardless).
func Run(env *fsx.Env) int {
	fmt.Fprintln(env.Stdout, env.Cwd)
	return exit.Ok
}
