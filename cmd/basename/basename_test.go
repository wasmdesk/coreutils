// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package basename

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"basename"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestBasenameSimple(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/usr/local/bin/foo", "foo\n"},
		{"foo", "foo\n"},
		{"/", "/\n"},
		{"a/b/", "b\n"},
		{"a///", "a\n"},
		{"", "\n"},
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.in)
		if rc := Run(e); rc != exit.Ok {
			t.Errorf("rc(%q) = %d", c.in, rc)
		}
		if got := out.String(); got != c.want {
			t.Errorf("basename %q = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBasenameSuffix(t *testing.T) {
	e, out, _ := newEnv("/a/b/foo.txt", ".txt")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "foo\n" {
		t.Errorf("got %q", got)
	}
}

// Suffix that does NOT match leaves the name alone.
func TestBasenameSuffixNoMatch(t *testing.T) {
	e, out, _ := newEnv("/a/foo.txt", ".md")
	_ = Run(e)
	if got := out.String(); got != "foo.txt\n" {
		t.Errorf("got %q", got)
	}
}

// Empty suffix is a no-op (matches GNU); name == suffix leaves the name alone.
func TestBasenameSuffixEdgeCases(t *testing.T) {
	e, out, _ := newEnv("/a/foo", "")
	_ = Run(e)
	if got := out.String(); got != "foo\n" {
		t.Errorf("empty suffix: got %q", got)
	}
	e2, out2, _ := newEnv("/a/foo", "foo")
	_ = Run(e2)
	if got := out2.String(); got != "foo\n" {
		t.Errorf("suffix == name: got %q", got)
	}
}

func TestBasenameNoArgs(t *testing.T) {
	e, _, errb := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestBasenameTooMany(t *testing.T) {
	e, _, errb := newEnv("a", "b", "c")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "too many") {
		t.Errorf("stderr = %q", errb.String())
	}
}

// Empty argv (Args[0] missing) is treated as zero operands.
func TestBasenameEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
