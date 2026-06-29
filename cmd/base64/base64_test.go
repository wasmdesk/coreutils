// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package base64

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
	return &fsx.Env{Args: append([]string{"base64"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestB64Encode(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("hello"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "aGVsbG8=\n" {
		t.Errorf("got %q", got)
	}
}

func TestB64Decode(t *testing.T) {
	e, out, _, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte("aGVsbG8="))
	_ = Run(e)
	if got := out.String(); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestB64DecodeWithWhitespace(t *testing.T) {
	e, out, _, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte("aGVs\nbG8=\n"))
	_ = Run(e)
	if got := out.String(); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestB64DecodeLong(t *testing.T) {
	e, out, _, m := newEnv("--decode", "/f")
	_ = m.WriteFile("/f", []byte("aGVsbG8="))
	_ = Run(e)
	if got := out.String(); got != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestB64DecodeBad(t *testing.T) {
	e, _, errb, m := newEnv("-d", "/f")
	_ = m.WriteFile("/f", []byte("###"))
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid input") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB64Stdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"base64"},
		Stdin:  strings.NewReader("hello"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "aGVsbG8=\n" {
		t.Errorf("got %q", got)
	}
}

func TestB64StdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"base64"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestB64StdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"base64"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestB64ExtraOperand(t *testing.T) {
	e, _, errb, _ := newEnv("/a", "/b")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "extra operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB64Missing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB64IsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestB64PrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
