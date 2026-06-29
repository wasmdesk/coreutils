// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package base32

import (
	stdbase32 "encoding/base32"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run encodes (default) or decodes (-d) its single file operand (or stdin)
// using standard base32. Encoded output ends with a newline.
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
				fmt.Fprintln(env.Stderr, "base32: extra operand: "+a)
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
			fmt.Fprintf(env.Stderr, "base32: stdin: %s\n", err)
			return exit.Fail
		}
		data = b
	} else {
		abs := fsx.Resolve(env.Cwd, path)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "base32: %s: %s\n", path, prettyErr(err))
			return exit.Fail
		}
		data = b
	}
	if decode {
		clean := strings.Map(func(r rune) rune {
			if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, string(data))
		out, err := stdbase32.StdEncoding.DecodeString(clean)
		if err != nil {
			fmt.Fprintf(env.Stderr, "base32: invalid input: %s\n", err)
			return exit.Fail
		}
		env.Stdout.Write(out)
		return exit.Ok
	}
	fmt.Fprintln(env.Stdout, stdbase32.StdEncoding.EncodeToString(data))
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
