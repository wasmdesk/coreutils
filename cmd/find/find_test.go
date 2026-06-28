// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package find

import (
	"bytes"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"find"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestFindAll(t *testing.T) {
	e, out, _, m := env(t, "/")
	_ = m.MkdirAll("/a/b")
	_ = m.WriteFile("/a/x", nil)
	_ = m.WriteFile("/a/b/y.txt", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	sort.Strings(got)
	want := []string{"/", "/a", "/a/b", "/a/b/y.txt", "/a/x"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFindName(t *testing.T) {
	e, out, _, m := env(t, "/", "-name", "*.txt")
	_ = m.MkdirAll("/a")
	_ = m.WriteFile("/a/x.txt", nil)
	_ = m.WriteFile("/a/y", nil)
	_ = m.WriteFile("/z.txt", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	sort.Strings(got)
	want := []string{"/a/x.txt", "/z.txt"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFindNoArgsUsesCwd(t *testing.T) {
	e, out, _, m := env(t)
	_ = m.WriteFile("/x", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if !strings.Contains(out.String(), "/x") {
		t.Errorf("out = %q", out.String())
	}
}

func TestFindEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

func TestFindNameNoValue(t *testing.T) {
	e, _, errb, _ := env(t, "-name")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestFindMissingRoot(t *testing.T) {
	e, _, errb, _ := env(t, "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestFindNamePatternBad(t *testing.T) {
	// An invalid glob (e.g. "[") should not crash; entries simply do not
	// match.
	e, out, _, m := env(t, "/", "-name", "[")
	_ = m.WriteFile("/x", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if out.Len() != 0 {
		t.Errorf("out = %q (want empty)", out.String())
	}
}

func TestFindOnFileRoot(t *testing.T) {
	e, out, _, m := env(t, "/f")
	_ = m.WriteFile("/f", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "/f\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestFindPrettyErr(t *testing.T) {
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

// Failing ReadDir mid-walk: prints an inner error line but the outer walk
// continues and the top-level rc remains Ok (only failure on the initial
// Stat propagates upward).
type rdFailFS struct {
	*fsx.MemFS
	failPath string
}

func (r *rdFailFS) ReadDir(p string) ([]fsx.FileInfo, error) {
	if p == r.failPath {
		return nil, fsx.ErrInvalid
	}
	return r.MemFS.ReadDir(p)
}

func TestFindInnerReadDirFails(t *testing.T) {
	m := fsx.NewMemFS()
	_ = m.MkdirAll("/a/sub")
	f := &rdFailFS{MemFS: m, failPath: "/a/sub"}
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: []string{"find", "/"}, Stdout: &out, Stderr: &errb, FS: f, Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "/a/sub") {
		t.Errorf("stderr = %q", errb.String())
	}
}

// The "matches root path" branch: name == "/" + havePattern returns false.
func TestFindNameSkipsRoot(t *testing.T) {
	e, out, _, m := env(t, "/", "-name", "*")
	_ = m.WriteFile("/x", nil)
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	// Root itself ("/") was skipped despite matching "*".
	if strings.Contains(out.String(), "/\n") {
		t.Errorf("root printed: %q", out.String())
	}
	if !strings.Contains(out.String(), "/x\n") {
		t.Errorf("/x missing: %q", out.String())
	}
}
