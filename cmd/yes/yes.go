// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package yes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints STRING (default "y") followed by a newline. Unlike GNU yes,
// which loops forever, we default to one line so the browser-hosted shell
// never wedges; -n COUNT raises the repeat count. STRING may be multiple
// words (joined by single spaces, matching GNU `yes hello world`).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	count := 1
	var words []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-n" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "yes: option requires an argument: -n")
				return exit.Usage
			}
			v, err := strconv.Atoi(args[i])
			if err != nil || v < 0 {
				fmt.Fprintf(env.Stderr, "yes: invalid count: %q\n", args[i])
				return exit.Usage
			}
			count = v
			continue
		}
		words = append(words, a)
	}
	msg := "y"
	if len(words) > 0 {
		msg = strings.Join(words, " ")
	}
	for i := 0; i < count; i++ {
		fmt.Fprintln(env.Stdout, msg)
	}
	return exit.Ok
}
