// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package envcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func stub(t *testing.T, env []string) {
	t.Helper()
	prev := Source
	Source = func() []string { return env }
	t.Cleanup(func() { Source = prev })
}

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"env"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestEnvSorted(t *testing.T) {
	stub(t, []string{"FOO=1", "BAR=2", "BAZ=3"})
	e, out, _ := newEnv()
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "BAR=2\nBAZ=3\nFOO=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestEnvEmpty(t *testing.T) {
	stub(t, nil)
	e, out, _ := newEnv()
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

// name=value operands, -i, -u: the printed environment.
func TestEnvAssignments(t *testing.T) {
	stub(t, []string{"A=1", "B=2", "C=3"})
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"X=1"}, "A=1\nB=2\nC=3\nX=1\n"},
		{[]string{"A=9", "Y=a=b"}, "A=9\nB=2\nC=3\nY=a=b\n"},
		{[]string{"-i", "X=1"}, "X=1\n"},
		{[]string{"-", "X=1"}, "X=1\n"},
		{[]string{"-u", "B", "-uC"}, "A=1\n"},
		{[]string{"--", "Z=0"}, "A=1\nB=2\nC=3\nZ=0\n"},
	} {
		e, out, errb := newEnv(c.args...)
		if rc := Run(e); rc != exit.Ok || out.String() != c.want {
			t.Errorf("env %q: %q rc %d %s", c.args, out.String(), rc, errb.String())
		}
	}
	for _, c := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-u"}, "requires an argument -- 'u'"},
		{[]string{"-z"}, "invalid option -- 'z'"},
	} {
		e, _, errb := newEnv(c.args...)
		if rc := Run(e); rc != exit.Usage || !strings.Contains(errb.String(), c.msg) {
			t.Errorf("env %q: rc %d %q", c.args, rc, errb.String())
		}
	}
}

// env CMD: the command, its environment and directory, its status.
func TestEnvUtility(t *testing.T) {
	stub(t, []string{"A=1", "PATH=/bin"})
	var got fsx.Command
	e, _, errb := newEnv("-u", "A", "PATH=/usr/bin", "tool", "x", "y")
	e.Cwd = "/w"
	e.Exec = func(c fsx.Command) (int, error) { got = c; return 5, nil }
	if rc := Run(e); rc != 5 {
		t.Errorf("rc %d %s", rc, errb.String())
	}
	if got.Name != "tool" || strings.Join(got.Args, ",") != "tool,x,y" || strings.Join(got.Env, ",") != "PATH=/usr/bin" || got.Dir != "/w" || got.Stdout != e.Stdout {
		t.Errorf("command %+v", got)
	}
	// -i with no assignment: an empty environment, not the caller's.
	e, _, _ = newEnv("-i", "tool")
	e.Exec = func(c fsx.Command) (int, error) { got = c; return 0, nil }
	if Run(e); got.Env == nil || len(got.Env) != 0 {
		t.Errorf("-i env %#v", got.Env)
	}
	// Not found, cannot run, and no Exec at all.
	e, _, errb = newEnv("nosuch")
	e.Exec = func(fsx.Command) (int, error) { return 0, fsx.ErrNoProgram }
	if rc := Run(e); rc != 127 || !strings.Contains(errb.String(), "env: nosuch: not found") {
		t.Errorf("127: %d %q", rc, errb.String())
	}
	e, _, errb = newEnv("tool")
	if rc := Run(e); rc != 126 || !strings.Contains(errb.String(), "cannot run programs here") {
		t.Errorf("126: %d %q", rc, errb.String())
	}
}

func TestEnvEmptyArgv(t *testing.T) {
	stub(t, nil)
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}
