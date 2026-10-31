// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build wasip1

package procx

import (
	"errors"
	"os"

	"github.com/go-wasmcontainers/shell/wasix/exec"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

var defaultExecer fsx.Execer = run

var stat = os.Stat

// run starts c through WASIX: <dir>/<name> or <dir>/<name>.wasm on PATH.
func run(c fsx.Command) (int, error) {
	p, err := exec.Find(stat, c.Dir, pathOf(c.Env), c.Name)
	if err != nil {
		return 0, fsx.ErrNoProgram
	}
	cmd := &exec.Cmd{Path: p, Args: c.Args, Env: c.Env, Dir: c.Dir, Stdin: c.Stdin, Stdout: c.Stdout, Stderr: c.Stderr}
	err = cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &ee):
		return ee.ExitCode(), nil
	case cmd.ProcessState != nil:
		return cmd.ProcessState.ExitCode(), nil // ran; a copy failed
	}
	return 0, err
}
