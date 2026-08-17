// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// OSFS implements FS over the real host filesystem. Native CLI wrappers
// (cmd/<tool>/main/main.go) construct one with NewOSFS() and let users speak
// real paths.
type OSFS struct{}

// NewOSFS returns the singleton OSFS handle. Stateless, so a single instance
// is enough; we expose a constructor for symmetry with NewMemFS.
func NewOSFS() *OSFS { return &OSFS{} }

// osReadDir is the directory-listing hook used for the post-Stat reads in
// ReadDir and Remove. It is a variable rather than a direct os.ReadDir call so
// that the TOCTOU / permission-denied error branches below are reachable in a
// test on every platform, including Windows, where the read-only attribute
// does not deny directory listing the way a POSIX chmod 0 does. Production code
// never reassigns it.
var osReadDir = os.ReadDir

// translate maps an os error onto our sentinels so tools can switch on
// fsx.ErrNotFound regardless of the backing store.
func translate(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotFound
	case errors.Is(err, fs.ErrExist):
		return ErrExists
	}
	return err
}

// Stat implements FS.
func (OSFS) Stat(p string) (FileInfo, error) {
	st, err := os.Stat(p)
	if err != nil {
		return FileInfo{}, translate(err)
	}
	return FileInfo{Name: st.Name(), Size: st.Size(), IsDir: st.IsDir()}, nil
}

// ReadFile implements FS.
func (OSFS) ReadFile(p string) ([]byte, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, translate(err)
	}
	if st.IsDir() {
		return nil, ErrIsDir
	}
	// Past the Stat gate the only failure left is a race (file vanished /
	// permission flipped between Stat and ReadFile). We forward whatever the
	// OS gives back through translate; no separate test path needed.
	b, err := os.ReadFile(p)
	return b, translate(err)
}

// WriteFile implements FS.
func (OSFS) WriteFile(p string, data []byte) error {
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return ErrIsDir
	}
	return translate(os.WriteFile(p, data, 0o644))
}

// Mkdir implements FS.
func (OSFS) Mkdir(p string) error { return translate(os.Mkdir(p, 0o755)) }

// MkdirAll implements FS.
func (OSFS) MkdirAll(p string) error { return translate(os.MkdirAll(p, 0o755)) }

// ReadDir implements FS.
func (OSFS) ReadDir(dir string) ([]FileInfo, error) {
	st, err := os.Stat(dir)
	if err != nil {
		return nil, translate(err)
	}
	if !st.IsDir() {
		return nil, ErrNotDir
	}
	// Past the Stat gate the only failure left is a TOCTOU race; we forward
	// it through translate without a parallel test path.
	entries, err := osReadDir(dir)
	if err != nil {
		return nil, translate(err)
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		info, ierr := e.Info()
		size := int64(0)
		if ierr == nil {
			size = info.Size()
		}
		out = append(out, FileInfo{Name: e.Name(), Size: size, IsDir: e.IsDir()})
	}
	sortByName(out)
	return out, nil
}

// Remove implements FS.
func (OSFS) Remove(p string) error {
	st, err := os.Stat(p)
	if err != nil {
		return translate(err)
	}
	if st.IsDir() {
		// On a dir, refuse if non-empty so callers must reach for RemoveAll.
		// os.ReadDir failures past the Stat gate are TOCTOU only; forwarded.
		entries, derr := osReadDir(p)
		if derr != nil {
			return translate(derr)
		}
		if len(entries) > 0 {
			return ErrNotEmpty
		}
	}
	return translate(os.Remove(p))
}

// RemoveAll implements FS. os.RemoveAll returns nil for missing paths,
// matching our contract.
func (OSFS) RemoveAll(p string) error { return translate(os.RemoveAll(p)) }

// Ensure filepath is referenced so the import is kept even when we drop the
// helper above (lint guard during refactors).
var _ = filepath.Separator
