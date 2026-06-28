// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package find

import (
	"errors"
	"fmt"
	"path"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run walks the tree rooted at ROOT (default cwd) and prints every path,
// optionally filtered by -name GLOB. The glob is shell-style (path.Match
// semantics: *, ?, [class]). Multiple ROOTs are allowed; their results are
// concatenated. Missing roots emit an error line but the walk continues.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var pattern string
	var havePattern bool
	var roots []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-name" {
			i++
			if i >= len(args) {
				fmt.Fprintln(env.Stderr, "find: option requires an argument: -name")
				return exit.Usage
			}
			pattern = args[i]
			havePattern = true
			continue
		}
		roots = append(roots, a)
	}
	if len(roots) == 0 {
		roots = []string{env.Cwd}
	}
	rc := exit.Ok
	for _, r := range roots {
		abs := fsx.Resolve(env.Cwd, r)
		if err := walk(env, abs, pattern, havePattern); err != nil {
			fmt.Fprintf(env.Stderr, "find: %s: %s\n", r, prettyErr(err))
			rc = exit.Fail
		}
	}
	return rc
}

// walk descends from p (depth-first, pre-order), printing matches.
func walk(env *fsx.Env, p string, pattern string, havePattern bool) error {
	info, err := env.FS.Stat(p)
	if err != nil {
		return err
	}
	if matches(info.Name, pattern, havePattern) {
		fmt.Fprintln(env.Stdout, p)
	}
	if !info.IsDir {
		return nil
	}
	entries, err := env.FS.ReadDir(p)
	if err != nil {
		return err
	}
	for _, e := range entries {
		// Skip-on-error for nested entries: a single inaccessible child
		// should not kill the whole walk; we just report and continue.
		child := fsx.Join(p, e.Name)
		if werr := walk(env, child, pattern, havePattern); werr != nil {
			fmt.Fprintf(env.Stderr, "find: %s: %s\n", child, prettyErr(werr))
		}
	}
	return nil
}

// matches reports whether the entry should print. When no pattern is set,
// every entry matches.
func matches(name, pattern string, havePattern bool) bool {
	if !havePattern {
		return true
	}
	// Special-case the root path: its Basename is "/" which never matches
	// a glob; skip.
	if name == "/" {
		return false
	}
	ok, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return ok
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
