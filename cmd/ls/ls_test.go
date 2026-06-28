// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ls

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
	return &fsx.Env{Args: append([]string{"ls"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestLsBasic(t *testing.T) {
	e, out, _, m := env(t)
	_ = m.Mkdir("/dir")
	_ = m.WriteFile("/a.txt", []byte("aa"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a.txt\ndir/\n" {
		t.Errorf("out = %q", got)
	}
}

func TestLsLong(t *testing.T) {
	e, out, _, m := env(t, "-l")
	_ = m.Mkdir("/dir")
	_ = m.WriteFile("/a.txt", []byte("aa"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	if !strings.Contains(got, "-        2 a.txt") || !strings.Contains(got, "d        0 dir") {
		t.Errorf("out = %q", got)
	}
}

func TestLsAllAccepted(t *testing.T) {
	e, _, _, _ := env(t, "-a")
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

func TestLsLAComposite(t *testing.T) {
	e, out, _, m := env(t, "-la")
	_ = m.WriteFile("/x", []byte("y"))
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(out.String(), "-        1 x") {
		t.Errorf("out = %q", out.String())
	}
	// -al alias
	e2, out2, _, _ := env(t, "-al")
	_ = e2.FS.(*fsx.MemFS).WriteFile("/x", []byte("y"))
	if rc := Run(e2); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(out2.String(), "-        1 x") {
		t.Errorf("out = %q", out2.String())
	}
}

func TestLsMissing(t *testing.T) {
	e, _, errb, _ := env(t, "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestLsMultiPath(t *testing.T) {
	e, out, _, m := env(t, "/a", "/b")
	_ = m.Mkdir("/a")
	_ = m.WriteFile("/a/x", nil)
	_ = m.Mkdir("/b")
	_ = m.WriteFile("/b/y", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	if !strings.Contains(got, "/a:\nx\n") || !strings.Contains(got, "\n/b:\ny\n") {
		t.Errorf("out = %q", got)
	}
}

func TestLsEmptyArgs(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

func TestLsPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
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

func TestLsOnFile(t *testing.T) {
	e, _, errb, m := env(t, "/f")
	_ = m.WriteFile("/f", nil)
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "not a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}
