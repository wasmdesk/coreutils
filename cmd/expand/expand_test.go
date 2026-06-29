// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package expand

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
	return &fsx.Env{Args: append([]string{"expand"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestExpandDefault(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("a\tb\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	// "a" at col 0, tab goes to col 8 -> 7 spaces, then "b\n".
	if got := out.String(); got != "a       b\n" {
		t.Errorf("got %q", got)
	}
}

func TestExpandTabstop(t *testing.T) {
	e, out, _, m := newEnv("-t", "4", "/f")
	_ = m.WriteFile("/f", []byte("ab\tcd\n"))
	_ = Run(e)
	// "ab" at col 0-1, tab -> col 4 -> 2 spaces, then "cd".
	if got := out.String(); got != "ab  cd\n" {
		t.Errorf("got %q", got)
	}
}

func TestExpandMultiTab(t *testing.T) {
	e, out, _, m := newEnv("-t", "2", "/f")
	_ = m.WriteFile("/f", []byte("\t\tX\n"))
	_ = Run(e)
	if got := out.String(); got != "    X\n" {
		t.Errorf("got %q", got)
	}
}

func TestExpandStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"expand", "-t", "4"},
		Stdin:  strings.NewReader("a\tb\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "a   b\n" {
		t.Errorf("got %q", got)
	}
}

func TestExpandStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"expand"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestExpandStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"expand"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestExpandDashTNoVal(t *testing.T) {
	e, _, errb, _ := newEnv("-t")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExpandDashTBad(t *testing.T) {
	for _, v := range []string{"foo", "0", "-1"} {
		e, _, errb, _ := newEnv("-t", v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid tabstop") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestExpandMissing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExpandIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExpandEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestExpandPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
