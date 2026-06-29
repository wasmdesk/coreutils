// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package rev

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run reverses each input line in place. UTF-8 aware: multi-byte runes stay
// intact (we reverse []rune, not []byte). Reads stdin when no paths are
// supplied.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var data string
	if len(args) == 0 {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "rev: stdin: %s\n", err)
			return exit.Fail
		}
		data = string(b)
	} else {
		var sb strings.Builder
		for _, p := range args {
			abs := fsx.Resolve(env.Cwd, p)
			b, err := env.FS.ReadFile(abs)
			if err != nil {
				fmt.Fprintf(env.Stderr, "rev: %s: %s\n", p, prettyErr(err))
				return exit.Fail
			}
			sb.Write(b)
		}
		data = sb.String()
	}
	for _, l := range splitLines(data) {
		fmt.Fprintln(env.Stdout, reverseRunes(l))
	}
	return exit.Ok
}

func reverseRunes(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
