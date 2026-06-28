// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package head

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
	return &fsx.Env{Args: append([]string{"head"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestHeadDefault10(t *testing.T) {
	e, out, _, m := env(t, "/f")
	body := ""
	for i := 1; i <= 15; i++ {
		body += "L\n"
		_ = i
	}
	_ = m.WriteFile("/f", []byte(body))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if strings.Count(out.String(), "\n") != 10 {
		t.Errorf("got %d lines, want 10: %q", strings.Count(out.String(), "\n"), out.String())
	}
}

func TestHeadN(t *testing.T) {
	e, out, _, m := env(t, "-n", "2", "/f")
	_ = m.WriteFile("/f", []byte("a\nb\nc\nd\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a\nb\n" {
		t.Errorf("out = %q", got)
	}
}

func TestHeadNoTrailingNL(t *testing.T) {
	e, out, _, m := env(t, "-n", "2", "/f")
	_ = m.WriteFile("/f", []byte("a\nb"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a\nb" {
		t.Errorf("out = %q", got)
	}
}

func TestHeadNZero(t *testing.T) {
	e, out, _, m := env(t, "-n", "0", "/f")
	_ = m.WriteFile("/f", []byte("a\nb\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if out.String() != "" {
		t.Errorf("out = %q", out.String())
	}
}

func TestHeadNoArgs(t *testing.T) {
	e, _, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestHeadEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestHeadDashNNoValue(t *testing.T) {
	e, _, errb, _ := env(t, "-n")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestHeadDashNBadValue(t *testing.T) {
	for _, v := range []string{"foo", "-1"} {
		e, _, errb, _ := env(t, "-n", v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid line count") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestHeadMissingFile(t *testing.T) {
	e, _, errb, _ := env(t, "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestHeadMulti(t *testing.T) {
	e, out, _, m := env(t, "/a", "/b")
	_ = m.WriteFile("/a", []byte("A\n"))
	_ = m.WriteFile("/b", []byte("B\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	if !strings.Contains(got, "==> /a <==\nA\n") || !strings.Contains(got, "==> /b <==\nB\n") {
		t.Errorf("out = %q", got)
	}
}

func TestHeadPrettyErr(t *testing.T) {
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
