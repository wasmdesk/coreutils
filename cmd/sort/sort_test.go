// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sort

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"sort"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestSortFile(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("b\na\nc\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a\nb\nc\n" {
		t.Errorf("got %q", got)
	}
}

func TestSortReverse(t *testing.T) {
	e, out, _, m := newEnv("-r", "/f")
	_ = m.WriteFile("/f", []byte("a\nb\nc\n"))
	_ = Run(e)
	if got := out.String(); got != "c\nb\na\n" {
		t.Errorf("got %q", got)
	}
}

func TestSortNumeric(t *testing.T) {
	e, out, _, m := newEnv("-n", "/f")
	_ = m.WriteFile("/f", []byte("10\n2\n30\n"))
	_ = Run(e)
	if got := out.String(); got != "2\n10\n30\n" {
		t.Errorf("got %q", got)
	}
}

// Numeric fallback: non-numeric lines compare lexicographically.
func TestSortNumericMixed(t *testing.T) {
	e, _, _, m := newEnv("-n", "/f")
	_ = m.WriteFile("/f", []byte("foo\nbar\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

func TestSortUnique(t *testing.T) {
	e, out, _, m := newEnv("-u", "/f")
	_ = m.WriteFile("/f", []byte("b\na\nb\nc\na\n"))
	_ = Run(e)
	if got := out.String(); got != "a\nb\nc\n" {
		t.Errorf("got %q", got)
	}
}

func TestSortStdin(t *testing.T) {
	m := fsx.NewMemFS()
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"sort"},
		Stdin:  strings.NewReader("y\nx\nz\n"),
		Stdout: &out,
		Stderr: new(bytes.Buffer),
		FS:     m,
		Cwd:    "/",
	}
	_ = Run(e)
	if got := out.String(); got != "x\ny\nz\n" {
		t.Errorf("got %q", got)
	}
}

func TestSortStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"sort"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
}

// io.ReadAll error path on stdin.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestSortStdinReadError(t *testing.T) {
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"sort"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "stdin") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSortMissingFile(t *testing.T) {
	e, _, errb, _ := newEnv("/missing")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSortIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/dir")
	_ = m.Mkdir("/dir")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSortMulti(t *testing.T) {
	e, out, _, m := newEnv("/a", "/b")
	_ = m.WriteFile("/a", []byte("3\n1\n"))
	_ = m.WriteFile("/b", []byte("2\n"))
	_ = Run(e)
	if got := out.String(); got != "1\n2\n3\n" {
		t.Errorf("got %q", got)
	}
}

func TestSortEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

// prettyErr branch -- exercise the non-fsx fallthrough.
func TestSortPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}

// dedup on empty slice.
func TestSortDedupEmpty(t *testing.T) {
	if got := dedup(nil); got != nil {
		t.Errorf("got %v", got)
	}
}

// Drive a stdin io.Reader path with empty content (splitLines == nil branch).
func TestSortStdinEmpty(t *testing.T) {
	e := &fsx.Env{Args: []string{"sort"}, Stdin: strings.NewReader(""), Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

var _ = io.EOF // keep the io import even if all tests drop direct refs
