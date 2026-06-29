// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sleep

import (
	"fmt"
	"strconv"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// SleepFn is the function used to sleep. Exposed so tests can swap in a
// no-op without actually blocking the test suite. Defaults to time.Sleep.
var SleepFn = time.Sleep

// Run pauses for the requested number of seconds (integer or floating-point).
// Multiple operands are summed (matching GNU `sleep 1 2 3` -> 6 seconds).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "sleep: missing operand")
		return exit.Usage
	}
	var total time.Duration
	for _, a := range args {
		v, err := strconv.ParseFloat(a, 64)
		if err != nil || v < 0 {
			fmt.Fprintf(env.Stderr, "sleep: invalid time interval: %q\n", a)
			return exit.Usage
		}
		total += time.Duration(v * float64(time.Second))
	}
	SleepFn(total)
	return exit.Ok
}
