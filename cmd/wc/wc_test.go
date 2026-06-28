// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package wc

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"wc"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestWcAll(t *testing.T) {
	e, out, _, m := env(t, "/f")
	_ = m.WriteFile("/f", []byte("hello world\nfoo bar baz\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	// 2 lines, 5 words, 24 bytes.
	if got, want := out.String(), "2 5 24 /f\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcL(t *testing.T) {
	e, out, _, m := env(t, "-l", "/f")
	_ = m.WriteFile("/f", []byte("a\nb\nc\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "3 /f\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcW(t *testing.T) {
	e, out, _, m := env(t, "-w", "/f")
	_ = m.WriteFile("/f", []byte("one two three"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "3 /f\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcC(t *testing.T) {
	e, out, _, m := env(t, "-c", "/f")
	_ = m.WriteFile("/f", []byte("hello"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "5 /f\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcLW(t *testing.T) {
	e, out, _, m := env(t, "-l", "-w", "/f")
	_ = m.WriteFile("/f", []byte("a b\nc d\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "2 4 /f\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcMulti(t *testing.T) {
	e, out, _, m := env(t, "-l", "/a", "/b")
	_ = m.WriteFile("/a", []byte("a\nb\n"))
	_ = m.WriteFile("/b", []byte("c\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	if !strings.Contains(got, "2 /a\n") || !strings.Contains(got, "1 /b\n") || !strings.Contains(got, "3 total\n") {
		t.Errorf("out = %q", got)
	}
}

func TestWcNoArgs(t *testing.T) {
	e, _, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestWcEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestWcMissing(t *testing.T) {
	e, _, errb, _ := env(t, "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestWcEmptyFile(t *testing.T) {
	e, out, _, m := env(t, "/e")
	_ = m.WriteFile("/e", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "0 0 0 /e\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestWcPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q", c.err, got)
		}
	}
}
