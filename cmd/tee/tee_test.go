// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tee

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(stdin io.Reader, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"tee"}, args...), Stdin: stdin, Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestTee(t *testing.T) {
	e, out, _, m := env(strings.NewReader("hello\n"), "/a", "b")
	if rc := Run(e); rc != exit.Ok || out.String() != "hello\n" {
		t.Fatalf("%d %q", rc, out.String())
	}
	for _, p := range []string{"/a", "/b"} {
		if b, _ := m.ReadFile(p); string(b) != "hello\n" {
			t.Errorf("%s = %q", p, b)
		}
	}
	e, _, _, m = env(strings.NewReader("two\n"), "-a", "/a")
	m.WriteFile("/a", []byte("one\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatal(rc)
	}
	if b, _ := m.ReadFile("/a"); string(b) != "one\ntwo\n" {
		t.Errorf("append: %q", b)
	}
	e, _, _, m = env(strings.NewReader("x"), "-ai", "--", "-f")
	if Run(e) != exit.Ok {
		t.Error("-- then a file named -f")
	}
	if b, _ := m.ReadFile("/-f"); string(b) != "x" {
		t.Errorf("-- file: %q", b)
	}
	e, _, _, _ = env(strings.NewReader(""), "-z")
	if Run(e) != exit.Usage {
		t.Error("bad option accepted")
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("broken") }

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestTeeErrors(t *testing.T) {
	e, _, errb, _ := env(failReader{})
	if Run(e) != exit.Fail || !strings.Contains(errb.String(), "broken") {
		t.Errorf("stdin error: %q", errb.String())
	}
	e, _, errb, _ = env(strings.NewReader("x"))
	e.Stdout = failWriter{}
	if Run(e) != exit.Fail || !strings.Contains(errb.String(), "closed") {
		t.Errorf("stdout error: %q", errb.String())
	}
	e, _, errb, m := env(strings.NewReader("x"), "/d")
	m.Mkdir("/d")
	if Run(e) != exit.Fail || errb.String() == "" {
		t.Error("writing a directory succeeded")
	}
	e, _, errb, m = env(strings.NewReader("x"), "-a", "/d")
	m.Mkdir("/d")
	if Run(e) != exit.Fail || errb.String() == "" {
		t.Error("appending to a directory succeeded")
	}
	e, _, _, _ = env(nil, "/n")
	if Run(e) != exit.Ok {
		t.Error("no stdin")
	}
	if Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS()}) != exit.Ok {
		t.Error("empty argv")
	}
}
