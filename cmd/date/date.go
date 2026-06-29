// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package date

import (
	"fmt"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Now is the clock seam. Defaults to time.Now; tests swap it in to render
// deterministic output without sneaking around the public Run signature.
var Now = time.Now

// Run prints the current time in UTC RFC1123 format by default. With -d
// STRING, it parses STRING as RFC3339 and re-prints that moment in UTC
// (handy as a round-trip helper without pulling in a strftime library).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var when time.Time
	when = Now().UTC()
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-d" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "date: option requires an argument: -d")
				return exit.Usage
			}
			t, err := time.Parse(time.RFC3339, args[i])
			if err != nil {
				fmt.Fprintf(env.Stderr, "date: invalid date: %q\n", args[i])
				return exit.Usage
			}
			when = t.UTC()
			continue
		}
		fmt.Fprintf(env.Stderr, "date: unknown argument: %q\n", a)
		return exit.Usage
	}
	fmt.Fprintln(env.Stdout, when.Format(time.RFC1123))
	return exit.Ok
}
