// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"errors"
	"io"
	"os"
)

// Opener is an optional FS capability: a file written as the data comes
// (tee watching a log, a long redirect), instead of all at once. OSFS and
// MemFS have it; Create falls back for an FS that does not.
type Opener interface {
	// Create opens p for writing: emptied (or created), or, with
	// appendMode, written after its current content. The errors are
	// WriteFile's: ErrNotFound for a missing parent, ErrIsDir, ...
	Create(p string, appendMode bool) (io.WriteCloser, error)
}

// Create opens p on fsys for writing as the data comes, through Opener
// when fsys has it. Otherwise the file is emptied at once (or, with
// appendMode, checked) and the data is written when the writer is closed.
func Create(fsys FS, p string, appendMode bool) (io.WriteCloser, error) {
	if o, ok := fsys.(Opener); ok {
		return o.Create(p, appendMode)
	}
	var old []byte
	if appendMode {
		b, err := fsys.ReadFile(p)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		old = b
	}
	if err := fsys.WriteFile(p, old); err != nil {
		return nil, err
	}
	return &wholeFile{fsys: fsys, p: p, data: old}, nil
}

// wholeFile buffers the data of an FS that can only write whole files.
type wholeFile struct {
	fsys FS
	p    string
	data []byte
}

func (w *wholeFile) Write(b []byte) (int, error) {
	w.data = append(w.data, b...)
	return len(b), nil
}

func (w *wholeFile) Close() error { return w.fsys.WriteFile(w.p, w.data) }

// Create implements Opener.
func (m *MemFS) Create(p string, appendMode bool) (io.WriteCloser, error) {
	c := Clean(p)
	n, ok := m.nodes[c]
	if !ok || !appendMode {
		if err := m.WriteFile(c, nil); err != nil {
			return nil, err
		}
		return &memWriter{n: m.nodes[c]}, nil
	}
	if n.isDir {
		return nil, ErrIsDir
	}
	return &memWriter{n: n}, nil
}

// memWriter appends to a node: what is written is there at once.
type memWriter struct{ n *memNode }

func (w *memWriter) Write(b []byte) (int, error) {
	w.n.data = append(w.n.data, b...)
	w.n.mtime = Now()
	return len(b), nil
}

func (w *memWriter) Close() error { return nil }

// Create implements Opener.
func (OSFS) Create(p string, appendMode bool) (io.WriteCloser, error) {
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return nil, ErrIsDir
	}
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if appendMode {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	f, err := os.OpenFile(p, flag, 0o644)
	if err != nil {
		return nil, translate(err)
	}
	return f, nil
}
