// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package base32

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

const helloB32 = "NBSWY3DP" // base32("hello")

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"base32"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestB32Encode(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("hello"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != helloB32+"\n" {
		t.Errorf("got %q", got)
	}
}

func TestB32Decode(t *testing.T) {
	e, out, _, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte(helloB32))
	_ = Run(e)
	if got := out.String(); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestB32DecodeLong(t *testing.T) {
	e, out, _, m := newEnv("--decode", "/f")
	_ = m.WriteFile("/f", []byte(helloB32+"\n"))
	_ = Run(e)
	if got := out.String(); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestB32DecodeBad(t *testing.T) {
	e, _, errb, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte("###"))
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid input") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB32Stdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"base32"},
		Stdin:  strings.NewReader("hello"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != helloB32+"\n" {
		t.Errorf("got %q", got)
	}
}

func TestB32StdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"base32"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestB32StdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"base32"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestB32ExtraOperand(t *testing.T) {
	e, _, errb, _ := newEnv("/a", "/b")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "extra operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB32Missing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB32IsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB32PrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
