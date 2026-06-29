// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package base64

import (
	stdbase64 "encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run encodes (default) or decodes (-d) its single file operand (or stdin)
// using standard base64. Encoded output ends with a newline; decoded output
// is byte-exact.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	decode := false
	var path string
	for _, a := range args {
		switch a {
		case "-d", "--decode":
			decode = true
		default:
			if path != "" {
				fmt.Fprintln(env.Stderr, "base64: extra operand: "+a)
				return exit.Usage
			}
			path = a
		}
	}
	var data []byte
	if path == "" {
		if env.Stdin == nil {
			return exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "base64: stdin: %s\n", err)
			return exit.Fail
		}
		data = b
	} else {
		abs := fsx.Resolve(env.Cwd, path)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "base64: %s: %s\n", path, prettyErr(err))
			return exit.Fail
		}
		data = b
	}
	if decode {
		// GNU base64 -d tolerates embedded whitespace.
		clean := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, string(data))
		out, err := stdbase64.StdEncoding.DecodeString(clean)
		if err != nil {
			fmt.Fprintf(env.Stderr, "base64: invalid input: %s\n", err)
			return exit.Fail
		}
		env.Stdout.Write(out)
		return exit.Ok
	}
	fmt.Fprintln(env.Stdout, stdbase64.StdEncoding.EncodeToString(data))
	return exit.Ok
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
