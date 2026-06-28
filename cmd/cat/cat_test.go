// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cat

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"cat"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestCatNoArgs(t *testing.T) {
	env, _, errb, _ := newEnv(t)
	if rc := Run(env); rc != exit.Usage {
		t.Errorf("rc = %d, want Usage", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCatBasic(t *testing.T) {
	env, out, _, m := newEnv(t, "/a.txt", "/b.txt")
	_ = m.WriteFile("/a.txt", []byte("alpha\n"))
	_ = m.WriteFile("/b.txt", []byte("beta\n"))
	if rc := Run(env); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "alpha\nbeta\n" {
		t.Errorf("out = %q", got)
	}
}

func TestCatMissing(t *testing.T) {
	env, out, errb, m := newEnv(t, "/a.txt", "/nope", "/b.txt")
	_ = m.WriteFile("/a.txt", []byte("A\n"))
	_ = m.WriteFile("/b.txt", []byte("B\n"))
	if rc := Run(env); rc != exit.Fail {
		t.Errorf("rc = %d, want Fail", rc)
	}
	if got := out.String(); got != "A\nB\n" {
		t.Errorf("stdout = %q", got)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCatNumbered(t *testing.T) {
	env, out, _, m := newEnv(t, "-n", "/x.txt")
	_ = m.WriteFile("/x.txt", []byte("one\ntwo\nthree"))
	if rc := Run(env); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	want := "     1\tone\n     2\ttwo\n     3\tthree"
	if got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestCatNumberedTrailingNL(t *testing.T) {
	env, out, _, m := newEnv(t, "-n", "/y.txt")
	_ = m.WriteFile("/y.txt", []byte("a\nb\n"))
	if rc := Run(env); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "     1\ta\n     2\tb\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestCatPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory"},
		{fsx.ErrNotDir, "not a directory"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}

func TestCatEmptyArgs(t *testing.T) {
	// Empty argv (Args == nil) -- exercises the len-check guard.
	env := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(env); rc != exit.Usage {
		t.Errorf("rc = %d, want Usage", rc)
	}
}

func TestCatIsDir(t *testing.T) {
	env, _, errb, m := newEnv(t, "/dir")
	_ = m.Mkdir("/dir")
	if rc := Run(env); rc != exit.Fail {
		t.Errorf("rc = %d, want Fail", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}
