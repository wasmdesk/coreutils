// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mv

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"mv"}, args...), Stdout: new(bytes.Buffer), Stderr: &errb, FS: m, Cwd: "/"}, &errb, m
}

func TestMvFileRename(t *testing.T) {
	e, _, m := env(t, "/a", "/b")
	_ = m.WriteFile("/a", []byte("hi"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if _, err := m.Stat("/a"); !errors.Is(err, fsx.ErrNotFound) {
		t.Errorf("/a still present")
	}
	if b, _ := m.ReadFile("/b"); string(b) != "hi" {
		t.Errorf("/b = %q", b)
	}
}

func TestMvFileIntoDir(t *testing.T) {
	e, _, m := env(t, "/a", "/d")
	_ = m.WriteFile("/a", []byte("hi"))
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/d/a"); string(b) != "hi" {
		t.Errorf("/d/a = %q", b)
	}
}

func TestMvMultiSrcMustBeDir(t *testing.T) {
	e, errb, m := env(t, "/a", "/b", "/dst")
	_ = m.WriteFile("/a", nil)
	_ = m.WriteFile("/b", nil)
	_ = m.WriteFile("/dst", nil)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "not a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMvMultiSrcIntoDir(t *testing.T) {
	e, _, m := env(t, "/a", "/b", "/d")
	_ = m.WriteFile("/a", []byte("A"))
	_ = m.WriteFile("/b", []byte("B"))
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/d/a"); string(b) != "A" {
		t.Errorf("d/a = %q", b)
	}
	if b, _ := m.ReadFile("/d/b"); string(b) != "B" {
		t.Errorf("d/b = %q", b)
	}
}

func TestMvDir(t *testing.T) {
	e, _, m := env(t, "/src", "/dst")
	_ = m.MkdirAll("/src/inner")
	_ = m.WriteFile("/src/x", []byte("X"))
	_ = m.WriteFile("/src/inner/y", []byte("Y"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/dst/inner/y"); string(b) != "Y" {
		t.Errorf("/dst/inner/y = %q", b)
	}
	if _, err := m.Stat("/src"); !errors.Is(err, fsx.ErrNotFound) {
		t.Errorf("/src still present")
	}
}

func TestMvNoArgs(t *testing.T) {
	e, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMvSrcMissing(t *testing.T) {
	e, errb, _ := env(t, "/nope", "/dst")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMvPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory"},
		{fsx.ErrNotDir, "not a directory"},
		{fsx.ErrExists, "file exists"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q", c.err, got)
		}
	}
}

// Failure injection covers the mv* deep-error branches.
type failFS struct {
	*fsx.MemFS
	failWrite, failRemove, failMkdirAll, failReadDir, failStat, failRead bool
}

func (f *failFS) WriteFile(p string, d []byte) error {
	if f.failWrite {
		return fsx.ErrInvalid
	}
	return f.MemFS.WriteFile(p, d)
}
func (f *failFS) Remove(p string) error {
	if f.failRemove {
		return fsx.ErrInvalid
	}
	return f.MemFS.Remove(p)
}
func (f *failFS) RemoveAll(p string) error {
	if f.failRemove {
		return fsx.ErrInvalid
	}
	return f.MemFS.RemoveAll(p)
}
func (f *failFS) MkdirAll(p string) error {
	if f.failMkdirAll {
		return fsx.ErrInvalid
	}
	return f.MemFS.MkdirAll(p)
}
func (f *failFS) ReadDir(p string) ([]fsx.FileInfo, error) {
	if f.failReadDir {
		return nil, fsx.ErrInvalid
	}
	return f.MemFS.ReadDir(p)
}
func (f *failFS) Stat(p string) (fsx.FileInfo, error) {
	if f.failStat {
		return fsx.FileInfo{}, fsx.ErrInvalid
	}
	return f.MemFS.Stat(p)
}
func (f *failFS) ReadFile(p string) ([]byte, error) {
	if f.failRead {
		return nil, fsx.ErrInvalid
	}
	return f.MemFS.ReadFile(p)
}

func TestMvWriteFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/a", []byte("x"))
	w := &failFS{MemFS: m, failWrite: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/a", "/b"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvRemoveFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/a", []byte("x"))
	w := &failFS{MemFS: m, failRemove: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/a", "/b"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvReadFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/a", []byte("x"))
	w := &failFS{MemFS: m, failRead: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/a", "/b"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvTreeMkdirAllFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	w := &failFS{MemFS: m, failMkdirAll: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvTreeReadDirFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	w := &failFS{MemFS: m, failReadDir: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

// Drive every inner mvTree branch: a dir with a sub-dir and a file inside.
// We trigger the inner-Stat-fail path with a custom FS that fails Stat only
// for a specific path (the sub-entry's child).
type pathFailFS struct {
	*fsx.MemFS
	failStatOn  string
	failReadOn  string
	failWriteOn string
	failMkdirOn string
}

func (p *pathFailFS) Stat(q string) (fsx.FileInfo, error) {
	if q == p.failStatOn {
		return fsx.FileInfo{}, fsx.ErrInvalid
	}
	return p.MemFS.Stat(q)
}
func (p *pathFailFS) ReadFile(q string) ([]byte, error) {
	if q == p.failReadOn {
		return nil, fsx.ErrInvalid
	}
	return p.MemFS.ReadFile(q)
}
func (p *pathFailFS) WriteFile(q string, d []byte) error {
	if q == p.failWriteOn {
		return fsx.ErrInvalid
	}
	return p.MemFS.WriteFile(q, d)
}
func (p *pathFailFS) MkdirAll(q string) error {
	if q == p.failMkdirOn {
		return fsx.ErrInvalid
	}
	return p.MemFS.MkdirAll(q)
}

func TestMvTreeInnerStatFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	_ = m.WriteFile("/s/x", []byte("X"))
	w := &pathFailFS{MemFS: m, failStatOn: "/s/x"}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvTreeInnerReadFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	_ = m.WriteFile("/s/x", []byte("X"))
	w := &pathFailFS{MemFS: m, failReadOn: "/s/x"}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvTreeInnerWriteFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	_ = m.WriteFile("/s/x", []byte("X"))
	w := &pathFailFS{MemFS: m, failWriteOn: "/d/x"}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestMvTreeInnerMkdirAllFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s/inner")
	w := &pathFailFS{MemFS: m, failMkdirOn: "/d/inner"}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"mv", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}
