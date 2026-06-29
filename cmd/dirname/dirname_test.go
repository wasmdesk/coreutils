// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package dirname

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"dirname"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestDirname(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/usr/local/bin/foo", "/usr/local/bin"},
		{"foo", "."},
		{"/", "/"},
		{"/foo", "/"},
		{"foo/bar", "foo"},
		{"a//b", "a"},
		{"a/", "."},
		{"", "."},
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.in)
		if rc := Run(e); rc != exit.Ok {
			t.Errorf("rc(%q) = %d", c.in, rc)
		}
		got := strings.TrimRight(out.String(), "\n")
		if got != c.want {
			t.Errorf("dirname %q = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDirnameMulti(t *testing.T) {
	e, out, _ := newEnv("/a/b", "/c/d/e")
	_ = Run(e)
	if got := out.String(); got != "/a\n/c/d\n" {
		t.Errorf("got %q", got)
	}
}

func TestDirnameNoArgs(t *testing.T) {
	e, _, errb := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestDirnameEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
