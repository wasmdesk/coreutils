// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package rm

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
	return &fsx.Env{Args: append([]string{"rm"}, args...), Stdout: new(bytes.Buffer), Stderr: &errb, FS: m, Cwd: "/"}, &errb, m
}

func TestRmFile(t *testing.T) {
	e, _, m := env(t, "/f")
	_ = m.WriteFile("/f", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if _, err := m.Stat("/f"); !errors.Is(err, fsx.ErrNotFound) {
		t.Errorf("file still present: %v", err)
	}
}

func TestRmMissing(t *testing.T) {
	e, errb, _ := env(t, "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRmMissingForce(t *testing.T) {
	e, errb, _ := env(t, "-f", "/nope")
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
	if errb.Len() != 0 {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRmNoArgs(t *testing.T) {
	e, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRmNoArgsForce(t *testing.T) {
	e, _, _ := env(t, "-f")
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

func TestRmEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestRmDirWithoutR(t *testing.T) {
	e, errb, m := env(t, "/d")
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRmRecursive(t *testing.T) {
	e, _, m := env(t, "-r", "/d")
	_ = m.MkdirAll("/d/sub")
	_ = m.WriteFile("/d/sub/x", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if _, err := m.Stat("/d"); !errors.Is(err, fsx.ErrNotFound) {
		t.Errorf("/d still present")
	}
}

func TestRmFlagsRcomposite(t *testing.T) {
	for _, flag := range []string{"-R", "-rf", "-fr", "-Rf", "-fR"} {
		e, _, m := env(t, flag, "/d")
		_ = m.MkdirAll("/d/inner")
		if rc := Run(e); rc != exit.Ok {
			t.Errorf("rc(%s) = %d", flag, rc)
		}
	}
}

func TestRmPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory"},
		{fsx.ErrNotEmpty, "directory not empty"},
		{fsx.ErrInvalid, "invalid argument"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q", c.err, got)
		}
	}
}

// The Remove-failure-with-force path: file exists per Stat, but Remove
// returns ErrNotEmpty because the file is in fact a non-empty dir without -r.
// Drive it by writing a tree, calling rm -f /d -- Stat says dir, then the
// "is a directory" branch fires; we instead drive Remove(failure)+force by
// passing a path that is a non-empty dir under -rf, then deleting one of its
// nodes mid-call... simpler: directly assert Run does not error when force is
// set AND RemoveAll returns no error, then drive the negative branch by
// pointing a removable dir at -r (no -f) with a sibling failure.
func TestRmContinuesOnError(t *testing.T) {
	e, errb, m := env(t, "/missing", "/exists")
	_ = m.WriteFile("/exists", nil)
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "/missing") {
		t.Errorf("stderr = %q", errb.String())
	}
	if _, err := m.Stat("/exists"); !errors.Is(err, fsx.ErrNotFound) {
		t.Errorf("/exists not removed")
	}
}

// Drive the "Remove returned error, force=true" silencing branch with a
// non-empty dir + -f (without -r): Stat passes, IsDir+!recursive fires --
// that's a different path. To hit the Remove-failure-with-force branch we
// need Remove() itself to fail post-Stat. Use a non-empty dir with -rf? No:
// -r switches to RemoveAll which won't fail. We need -f without -r on a
// non-empty dir; but that takes the IsDir+!recursive branch. The remaining
// Remove-failure path is structurally unreachable in our MemFS unless we
// inject a faulty FS. Cover it with a small stub.
type errFS struct{ *fsx.MemFS }

func (e *errFS) Remove(p string) error    { return fsx.ErrInvalid }
func (e *errFS) RemoveAll(p string) error { return fsx.ErrInvalid }

func TestRmRemoveFailsForce(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/f", nil)
	fs := &errFS{m}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"rm", "-f", "/f"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: fs, Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
	if errb.Len() != 0 {
		t.Errorf("stderr = %q (force should silence)", errb.String())
	}
}

func TestRmRemoveFailsNoForce(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.WriteFile("/f", nil)
	fs := &errFS{m}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"rm", "/f"}, Stdout: new(bytes.Buffer), Stderr: &errb, FS: fs, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}
