// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package mkdir

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"mkdir"}, args...), Stdout: new(bytes.Buffer), Stderr: &errb, FS: m, Cwd: "/"}, &errb, m
}

func TestMkdirBasic(t *testing.T) {
	e, _, m := env(t, "/a")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if info, _ := m.Stat("/a"); !info.IsDir {
		t.Errorf("/a not dir")
	}
}

func TestMkdirNoArgs(t *testing.T) {
	e, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMkdirEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestMkdirExisting(t *testing.T) {
	e, errb, m := env(t, "/a")
	_ = m.Mkdir("/a")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "file exists") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMkdirParents(t *testing.T) {
	e, _, m := env(t, "-p", "/a/b/c")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if info, _ := m.Stat("/a/b/c"); !info.IsDir {
		t.Errorf("/a/b/c not dir")
	}
}

func TestMkdirMissingParent(t *testing.T) {
	e, errb, _ := env(t, "/x/y")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestMkdirPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrExists, "file exists"},
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrNotDir, "not a directory"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q", c.err, got)
		}
	}
}
