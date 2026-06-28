// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cp

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
	return &fsx.Env{Args: append([]string{"cp"}, args...), Stdout: new(bytes.Buffer), Stderr: &errb, FS: m, Cwd: "/"}, &errb, m
}

func TestCpFileToFile(t *testing.T) {
	e, _, m := env(t, "/a", "/b")
	_ = m.WriteFile("/a", []byte("hi"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/b"); string(b) != "hi" {
		t.Errorf("b = %q", b)
	}
}

func TestCpFileIntoDir(t *testing.T) {
	e, _, m := env(t, "/a", "/d")
	_ = m.WriteFile("/a", []byte("hi"))
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/d/a"); string(b) != "hi" {
		t.Errorf("d/a = %q", b)
	}
}

func TestCpMultiSrcMustBeDir(t *testing.T) {
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

func TestCpMultiSrcIntoDir(t *testing.T) {
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

func TestCpRecursive(t *testing.T) {
	e, _, m := env(t, "-r", "/src", "/dst")
	_ = m.MkdirAll("/src/inner")
	_ = m.WriteFile("/src/x", []byte("X"))
	_ = m.WriteFile("/src/inner/y", []byte("Y"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/dst/x"); string(b) != "X" {
		t.Errorf("dst/x = %q", b)
	}
	if b, _ := m.ReadFile("/dst/inner/y"); string(b) != "Y" {
		t.Errorf("dst/inner/y = %q", b)
	}
}

func TestCpRecursiveBigR(t *testing.T) {
	e, _, m := env(t, "-R", "/s", "/d")
	_ = m.MkdirAll("/s")
	_ = m.WriteFile("/s/x", []byte("X"))
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
	if b, _ := m.ReadFile("/d/x"); string(b) != "X" {
		t.Errorf("d/x = %q", b)
	}
}

func TestCpNoArgs(t *testing.T) {
	e, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCpEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestCpSrcMissing(t *testing.T) {
	e, errb, _ := env(t, "/nope", "/dst")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCpDirWithoutR(t *testing.T) {
	e, errb, m := env(t, "/s", "/d")
	_ = m.Mkdir("/s")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCpPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory (use -r)"},
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

// errFS fails MkdirAll + WriteFile to drive the recursive copy's failure
// branches.
type writeFS struct {
	*fsx.MemFS
	failMkdirAll bool
	failWrite    bool
	failReadDir  bool
}

func (w *writeFS) MkdirAll(p string) error {
	if w.failMkdirAll {
		return fsx.ErrInvalid
	}
	return w.MemFS.MkdirAll(p)
}
func (w *writeFS) WriteFile(p string, d []byte) error {
	if w.failWrite {
		return fsx.ErrInvalid
	}
	return w.MemFS.WriteFile(p, d)
}
func (w *writeFS) ReadDir(p string) ([]fsx.FileInfo, error) {
	if w.failReadDir {
		return nil, fsx.ErrInvalid
	}
	return w.MemFS.ReadDir(p)
}

func TestCpRecursiveMkdirFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	w := &writeFS{MemFS: m, failMkdirAll: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"cp", "-r", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestCpFileWriteFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/s", []byte("x"))
	w := &writeFS{MemFS: m, failWrite: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"cp", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestCpRecursiveReadDirFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	w := &writeFS{MemFS: m, failReadDir: true}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"cp", "-r", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

// Failure inside a nested copyAny (e.g. inner ReadFile fails) propagates
// through copyTree. We trigger it by making one entry a file that ReadFile
// will reject by returning ErrInvalid via a wrapper.
type innerErrFS struct{ *fsx.MemFS }

func (e *innerErrFS) ReadFile(p string) ([]byte, error) { return nil, fsx.ErrInvalid }

func TestCpRecursiveInnerFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/s")
	_ = m.WriteFile("/s/x", []byte("X"))
	w := &innerErrFS{m}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"cp", "-r", "/s", "/d"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: w, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}
