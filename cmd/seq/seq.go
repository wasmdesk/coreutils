// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package seq

import (
	"fmt"
	"strconv"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints an integer sequence. Forms:
//
//	seq N        -> 1, 2, ..., N
//	seq A B      -> A, A+1, ..., B   (step 1; empty if A > B)
//	seq A S B    -> A, A+S, ..., <=B (or >=B when S < 0)
//
// A zero step or a mismatched-direction step is rejected as Usage.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 || len(args) > 3 {
		fmt.Fprintln(env.Stderr, "seq: usage: seq [FIRST [INCREMENT]] LAST")
		return exit.Usage
	}
	parsed := make([]int, len(args))
	for i, a := range args {
		v, err := strconv.Atoi(a)
		if err != nil {
			fmt.Fprintf(env.Stderr, "seq: invalid integer: %q\n", a)
			return exit.Usage
		}
		parsed[i] = v
	}
	var first, step, last int
	switch len(parsed) {
	case 1:
		first, step, last = 1, 1, parsed[0]
	case 2:
		first, step, last = parsed[0], 1, parsed[1]
	case 3:
		first, step, last = parsed[0], parsed[1], parsed[2]
	}
	if step == 0 {
		fmt.Fprintln(env.Stderr, "seq: invalid Zero increment value: 0")
		return exit.Usage
	}
	if step > 0 {
		for v := first; v <= last; v += step {
			fmt.Fprintln(env.Stdout, v)
		}
	} else {
		for v := first; v >= last; v += step {
			fmt.Fprintln(env.Stdout, v)
		}
	}
	return exit.Ok
}
