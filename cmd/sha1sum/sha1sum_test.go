// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sha1sum

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

const sha1OfHello = "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"sha1sum"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestSha1File(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("hello"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != sha1OfHello+"  /f\n" {
		t.Errorf("got %q", got)
	}
}

func TestSha1Stdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"sha1sum"},
		Stdin:  strings.NewReader("hello"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != sha1OfHello+"  -\n" {
		t.Errorf("got %q", got)
	}
}

func TestSha1StdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"sha1sum"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestSha1StdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"sha1sum"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestSha1Multi(t *testing.T) {
	e, out, _, m := newEnv("/a", "/b")
	_ = m.WriteFile("/a", []byte("hello"))
	_ = m.WriteFile("/b", []byte("world"))
	_ = Run(e)
	got := out.String()
	if !strings.Contains(got, "  /a\n") || !strings.Contains(got, "  /b\n") {
		t.Errorf("got %q", got)
	}
}

func TestSha1Missing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSha1IsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSha1PrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
