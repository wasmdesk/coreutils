// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package rev

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"rev"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestRevFile(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("abc\ndef\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "cba\nfed\n" {
		t.Errorf("got %q", got)
	}
}

func TestRevUTF8(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("héllo\n"))
	_ = Run(e)
	if got := out.String(); got != "olléh\n" {
		t.Errorf("got %q", got)
	}
}

func TestRevStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"rev"},
		Stdin:  strings.NewReader("hello\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "olleh\n" {
		t.Errorf("got %q", got)
	}
}

func TestRevStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"rev"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestRevStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"rev"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestRevMissing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRevIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRevEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestRevPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
