// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package uniq

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
	return &fsx.Env{Args: append([]string{"uniq"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestUniqDefault(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("a\na\nb\nc\nc\nc\na\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a\nb\nc\na\n" {
		t.Errorf("got %q", got)
	}
}

func TestUniqCount(t *testing.T) {
	e, out, _, m := newEnv("-c", "/f")
	_ = m.WriteFile("/f", []byte("a\na\nb\n"))
	_ = Run(e)
	if got := out.String(); got != "      2 a\n      1 b\n" {
		t.Errorf("got %q", got)
	}
}

func TestUniqDups(t *testing.T) {
	e, out, _, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte("a\na\nb\nc\nc\n"))
	_ = Run(e)
	if got := out.String(); got != "a\nc\n" {
		t.Errorf("got %q", got)
	}
}

func TestUniqUniques(t *testing.T) {
	e, out, _, m := newEnv("-u", "/f")
	_ = m.WriteFile("/f", []byte("a\na\nb\nc\nc\n"))
	_ = Run(e)
	if got := out.String(); got != "b\n" {
		t.Errorf("got %q", got)
	}
}

func TestUniqStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"uniq"},
		Stdin:  strings.NewReader("x\nx\ny\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "x\ny\n" {
		t.Errorf("got %q", got)
	}
}

func TestUniqStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"uniq"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestUniqStdinReadError(t *testing.T) {
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"uniq"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "stdin") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUniqMissingFile(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUniqIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUniqExtraOperand(t *testing.T) {
	e, _, errb, _ := newEnv("/a", "/b")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "extra operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUniqEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestUniqPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
