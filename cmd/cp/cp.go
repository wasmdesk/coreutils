// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cp

import (
	"errors"
	"fmt"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run copies SRC to DST. With -r (or -R) directories are copied recursively.
// If DST is an existing directory, the source's basename is appended.
// Multiple SRCs are allowed only when DST is a directory.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	recursive := false
	var pos []string
	for _, a := range args {
		switch a {
		case "-r", "-R":
			recursive = true
		default:
			pos = append(pos, a)
		}
	}
	if len(pos) < 2 {
		fmt.Fprintln(env.Stderr, "cp: missing operand")
		return exit.Usage
	}
	srcs, dst := pos[:len(pos)-1], pos[len(pos)-1]
	absDst := fsx.Resolve(env.Cwd, dst)
	dstInfo, dstErr := env.FS.Stat(absDst)
	dstIsDir := dstErr == nil && dstInfo.IsDir
	if len(srcs) > 1 && !dstIsDir {
		fmt.Fprintf(env.Stderr, "cp: target %q is not a directory\n", dst)
		return exit.Usage
	}
	rc := exit.Ok
	for _, s := range srcs {
		absSrc := fsx.Resolve(env.Cwd, s)
		target := absDst
		if dstIsDir {
			target = fsx.Join(absDst, fsx.Basename(absSrc))
		}
		if err := copyAny(env.FS, absSrc, target, recursive); err != nil {
			fmt.Fprintf(env.Stderr, "cp: %s: %s\n", s, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

// copyAny copies src->dst, descending if src is a dir and recursive is set.
func copyAny(fs fsx.FS, src, dst string, recursive bool) error {
	info, err := fs.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir {
		if !recursive {
			return fsx.ErrIsDir
		}
		return copyTree(fs, src, dst)
	}
	data, err := fs.ReadFile(src)
	if err != nil {
		return err
	}
	return fs.WriteFile(dst, data)
}

// copyTree mirrors the src dir at dst, recursing into subdirectories.
func copyTree(fs fsx.FS, src, dst string) error {
	if err := fs.MkdirAll(dst); err != nil {
		return err
	}
	entries, err := fs.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := copyAny(fs, fsx.Join(src, e.Name), fsx.Join(dst, e.Name), true); err != nil {
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
		return "is a directory (use -r)"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	case errors.Is(err, fsx.ErrExists):
		return "file exists"
	}
	return err.Error()
}
