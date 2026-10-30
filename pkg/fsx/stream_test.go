// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"errors"
	"path/filepath"
	"testing"
)

// plain hides MemFS's Opener (only FS's methods are promoted): the
// fallback path.
type plain struct{ FS }

// readFails is an FS, without Opener, whose files cannot be read.
type readFails struct{ FS }

func (readFails) ReadFile(string) ([]byte, error) { return nil, errors.New("eio") }

func streamCases(t *testing.T, name string, fsys FS, dir string, live bool) {
	t.Run(name, func(t *testing.T) {
		p := dir + "/f"
		read := func() string {
			b, err := fsys.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			return string(b)
		}
		if err := fsys.WriteFile(p, []byte("old\n")); err != nil {
			t.Fatal(err)
		}
		w, err := Create(fsys, p, false)
		if err != nil {
			t.Fatal(err)
		}
		if s := read(); s != "" {
			t.Errorf("not emptied at open: %q", s)
		}
		w.Write([]byte("a\n"))
		if s := read(); live && s != "a\n" {
			t.Errorf("not written as it comes: %q", s)
		}
		if err := w.Close(); err != nil || read() != "a\n" {
			t.Errorf("close: %v %q", err, read())
		}
		w, err = Create(fsys, p, true)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("b\n"))
		if s := read(); live && s != "a\nb\n" {
			t.Errorf("append, live: %q", s)
		}
		w.Close()
		if s := read(); s != "a\nb\n" {
			t.Errorf("append: %q", s)
		}
		// Append to a file that does not exist creates it.
		w, err = Create(fsys, dir+"/new", true)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("n"))
		w.Close()
		if b, _ := fsys.ReadFile(dir + "/new"); string(b) != "n" {
			t.Errorf("new: %q", b)
		}
		for _, mode := range []bool{false, true} {
			if _, err := Create(fsys, dir, mode); !errors.Is(err, ErrIsDir) {
				t.Errorf("a directory (append %v): %v", mode, err)
			}
			if _, err := Create(fsys, dir+"/nodir/f", mode); !errors.Is(err, ErrNotFound) {
				t.Errorf("a missing parent (append %v): %v", mode, err)
			}
		}
	})
}

func TestCreate(t *testing.T) {
	m := NewMemFS()
	m.Mkdir("/d")
	streamCases(t, "MemFS", m, "/d", true)
	p := NewMemFS()
	p.Mkdir("/d")
	streamCases(t, "fallback", plain{p}, "/d", false)
	streamCases(t, "OSFS", NewOSFS(), filepath.ToSlash(t.TempDir()), true)

	r := NewMemFS()
	r.WriteFile("/f", []byte("x"))
	if _, err := Create(readFails{r}, "/f", true); err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("fallback append over an unreadable file: %v", err)
	}
}
