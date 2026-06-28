// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package fsx is the abstract filesystem + execution environment every
// coreutils tool speaks. A tool's signature is:
//
//	func Run(env *fsx.Env) int
//
// where env carries argv (already split, env.Args[0] is the tool name), the
// three stdio streams, the current working directory, and a pluggable FS the
// tool reads/writes through. The same Run function powers both shipping modes:
//
//   - native CLI: cmd/<tool>/main/main.go wires env.FS=OSFS, env.Stdin=os.Stdin,
//     env.Cwd=os.Getwd(), env.Args=os.Args, and os.Exit(Run(env)).
//
//   - in-browser builtin: the wasmbox terminal wraps its sharedvfs.VFS into a
//     fsx.FS adapter, pipes the terminal's output []string<->bytes.Buffer, and
//     hands the env to coreutils.Dispatch(name, env).
//
// The FS contract is intentionally narrow (Open / Stat / List / Mkdir / Write /
// Remove + a few helpers). Tools that need richer behaviour (e.g. cp -r) walk
// the tree themselves using the supplied primitives -- this keeps every host
// backend trivial to implement and lets a 100-tool roadmap reuse one FS.
package fsx

import (
	"errors"
	"io"
	"sort"
	"strings"
)

// FileInfo describes one entry. Mirrors os.FileInfo's load-bearing fields
// without dragging the os package in (so wasm hosts can implement FS without
// linking syscall/js's stat machinery).
type FileInfo struct {
	Name  string // basename
	Size  int64  // length of file content; 0 for directories
	IsDir bool
}

// FS is the storage contract every tool talks to. Paths are POSIX-style
// (forward-slash separators, "/" is root). Implementations MUST Clean paths
// before resolving so "a//b/../c" and "a/c" hit the same node.
type FS interface {
	// Stat returns the FileInfo for p, or ErrNotFound.
	Stat(p string) (FileInfo, error)

	// ReadFile returns the full file body. Returns ErrIsDir on a directory.
	ReadFile(p string) ([]byte, error)

	// WriteFile replaces (or creates) the file at p with data. Parent must
	// exist; the call does NOT auto-mkdir. ErrIsDir if p is a directory.
	WriteFile(p string, data []byte) error

	// Mkdir creates a single directory at p. Parent must exist. ErrExists if
	// the path is already occupied.
	Mkdir(p string) error

	// MkdirAll creates p plus any missing parents (mkdir -p). Returns nil if
	// p already exists as a directory.
	MkdirAll(p string) error

	// ReadDir returns the immediate children of dir, sorted alphabetically.
	// ErrNotFound for missing paths, ErrNotDir for files.
	ReadDir(dir string) ([]FileInfo, error)

	// Remove deletes a single file or an EMPTY directory. ErrNotEmpty if dir
	// is non-empty.
	Remove(p string) error

	// RemoveAll deletes p plus all descendants (rm -rf). Removing a missing
	// path is a no-op (returns nil) -- matches GNU rm -f semantics.
	RemoveAll(p string) error
}

// Sentinel errors. Kept as exported vars so callers can `errors.Is` without
// parsing message strings.
var (
	ErrNotFound = errors.New("fsx: not found")
	ErrNotDir   = errors.New("fsx: not a directory")
	ErrIsDir    = errors.New("fsx: is a directory")
	ErrExists   = errors.New("fsx: already exists")
	ErrNotEmpty = errors.New("fsx: directory not empty")
	ErrInvalid  = errors.New("fsx: invalid argument")
)

// Env is the per-invocation execution environment. Every tool's Run reads
// argv from Args (Args[0] is conventionally the tool name, mirroring os.Args),
// reads stdin from Stdin, writes to Stdout/Stderr, resolves bare paths
// against Cwd, and operates on FS.
type Env struct {
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Cwd    string
	FS     FS
}

// Clean is a POSIX-only path cleaner that does NOT depend on path/filepath
// (so it behaves identically on every host). Collapses repeated slashes,
// resolves "." and "..", and returns either "/" or a leading-slash path with
// no trailing slash.
func Clean(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			// skip
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

// Join combines dir + name into a Clean absolute path.
func Join(dir, name string) string {
	if dir == "" {
		dir = "/"
	}
	if dir == "/" {
		return Clean("/" + name)
	}
	return Clean(dir + "/" + name)
}

// Parent returns the parent directory of p. The parent of "/" is "/" so
// repeated calls converge.
func Parent(p string) string {
	c := Clean(p)
	if c == "/" {
		return "/"
	}
	i := strings.LastIndex(c, "/")
	if i <= 0 {
		return "/"
	}
	return c[:i]
}

// Basename returns the trailing component of p ("/" -> "/").
func Basename(p string) string {
	c := Clean(p)
	if c == "/" {
		return "/"
	}
	i := strings.LastIndex(c, "/")
	return c[i+1:]
}

// Resolve makes a path absolute against cwd. Absolute paths are returned
// as-is (Cleaned); relative paths are joined onto cwd.
func Resolve(cwd, p string) string {
	if p == "" {
		return Clean(cwd)
	}
	if strings.HasPrefix(p, "/") {
		return Clean(p)
	}
	return Join(cwd, p)
}

// sortByName sorts a slice of FileInfo alphabetically. Pulled out so both
// MemFS and OSFS produce identical orderings without each rolling their own.
func sortByName(es []FileInfo) {
	sort.SliceStable(es, func(i, j int) bool {
		return es[i].Name < es[j].Name
	})
}
