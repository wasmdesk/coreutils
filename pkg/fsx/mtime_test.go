// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemFSModTime(t *testing.T) {
	defer func(n func() time.Time) { Now = n }(Now)
	t0 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	Now = func() time.Time { return t0 }
	m := NewMemFS()
	m.Mkdir("/d")
	m.MkdirAll("/e/f")
	m.WriteFile("/d/x", []byte("1"))
	for _, p := range []string{"/", "/d", "/e", "/e/f", "/d/x"} {
		if st, _ := m.Stat(p); !st.ModTime.Equal(t0) {
			t.Errorf("%s: %v", p, st.ModTime)
		}
	}
	t1 := t0.Add(time.Hour)
	Now = func() time.Time { return t1 }
	w, _ := m.Create("/d/x", true)
	w.Write([]byte("2"))
	if es, _ := m.ReadDir("/d"); !es[0].ModTime.Equal(t1) {
		t.Errorf("a streamed write: %v", es[0].ModTime)
	}
	t2 := t0.Add(-48 * time.Hour)
	if err := m.Chtimes("/d/x", t2); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.Stat("/d/x"); !st.ModTime.Equal(t2) {
		t.Errorf("chtimes: %v", st.ModTime)
	}
	if err := m.Chtimes("/nope", t2); !errors.Is(err, ErrNotFound) {
		t.Errorf("chtimes missing: %v", err)
	}
}

func TestOSFSModTime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	os.WriteFile(p, []byte("x"), 0o644)
	want := time.Date(2025, 5, 6, 7, 8, 9, 0, time.UTC)
	if err := os.Chtimes(p, want, want); err != nil {
		t.Fatal(err)
	}
	if st, _ := NewOSFS().Stat(p); !st.ModTime.Equal(want) {
		t.Errorf("stat: %v", st.ModTime)
	}
	if es, _ := NewOSFS().ReadDir(dir); len(es) != 1 || !es[0].ModTime.Equal(want) {
		t.Errorf("readdir: %+v", es)
	}
}
