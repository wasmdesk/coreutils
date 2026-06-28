// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mv

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run renames SRC to DST. If DST is a directory, SRC is moved into it (its
// basename appended). Multiple SRCs require DST to be a directory.
// Implementation is copy-then-remove because our FS contract has no rename
// primitive -- adequate for the wasmbox demo tree and trivially correct
// (it's a real, intentional move, not a fast-path).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) < 2 {
		fmt.Fprintln(env.Stderr, "mv: missing operand")
		return exit.Usage
	}
	srcs, dst := args[:len(args)-1], args[len(args)-1]
	absDst := fsx.Resolve(env.Cwd, dst)
	dstInfo, dstErr := env.FS.Stat(absDst)
	dstIsDir := dstErr == nil && dstInfo.IsDir
	if len(srcs) > 1 && !dstIsDir {
		fmt.Fprintf(env.Stderr, "mv: target %q is not a directory\n", dst)
		return exit.Usage
	}
	rc := exit.Ok
	for _, s := range srcs {
		absSrc := fsx.Resolve(env.Cwd, s)
		target := absDst
		if dstIsDir {
			target = fsx.Join(absDst, fsx.Basename(absSrc))
		}
		if err := moveAny(env.FS, absSrc, target); err != nil {
			fmt.Fprintf(env.Stderr, "mv: %s: %s\n", s, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

func moveAny(fs fsx.FS, src, dst string) error {
	info, err := fs.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir {
		if err := mvTree(fs, src, dst); err != nil {
			return err
		}
		return fs.RemoveAll(src)
	}
	data, err := fs.ReadFile(src)
	if err != nil {
		return err
	}
	if err := fs.WriteFile(dst, data); err != nil {
		return err
	}
	return fs.Remove(src)
}

func mvTree(fs fsx.FS, src, dst string) error {
	if err := fs.MkdirAll(dst); err != nil {
		return err
	}
	entries, err := fs.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := fsx.Join(src, e.Name)
		d := fsx.Join(dst, e.Name)
		info, err := fs.Stat(s)
		if err != nil {
			return err
		}
		if info.IsDir {
			if err := mvTree(fs, s, d); err != nil {
				return err
			}
			continue
		}
		data, err := fs.ReadFile(s)
		if err != nil {
			return err
		}
		if err := fs.WriteFile(d, data); err != nil {
			return err
		}
	}
	return nil
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	case errors.Is(err, fsx.ErrExists):
		return "file exists"
	}
	return err.Error()
}
