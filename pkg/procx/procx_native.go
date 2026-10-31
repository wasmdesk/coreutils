// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build !wasip1 && !js

package procx

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

var defaultExecer fsx.Execer = run

// lookPath finds name on path (each element tried with exec.LookPath,
// which checks the file is executable), or uses it as is with a slash.
func lookPath(path, dir, name string) (string, error) {
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, filepath.Separator) {
		if !filepath.IsAbs(name) {
			name = filepath.Join(dir, name)
		}
		return exec.LookPath(name)
	}
	for _, d := range filepath.SplitList(path) {
		if d == "" {
			d = "."
		}
		if !filepath.IsAbs(d) {
			d = filepath.Join(dir, d)
		}
		if p, err := exec.LookPath(filepath.Join(d, name)); err == nil {
			return p, nil
		}
	}
	return "", exec.ErrNotFound
}

// run starts c through os/exec.
func run(c fsx.Command) (int, error) {
	p, err := lookPath(pathOf(c.Env), c.Dir, c.Name)
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
		return status(ee.ExitCode()), nil
	case cmd.ProcessState != nil:
		return status(cmd.ProcessState.ExitCode()), nil // ran; a copy failed
	}
	return 0, err
}
