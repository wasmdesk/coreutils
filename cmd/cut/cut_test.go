// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cut

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"cut"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestCutCharsSingle(t *testing.T) {
	e, out, _, m := newEnv("-c", "1", "/f")
	_ = m.WriteFile("/f", []byte("abc\ndef\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "a\nd\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutCharsRange(t *testing.T) {
	e, out, _, m := newEnv("-c", "1-2", "/f")
	_ = m.WriteFile("/f", []byte("abcdef\n"))
	_ = Run(e)
	if got := out.String(); got != "ab\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutCharsOpenRange(t *testing.T) {
	e, out, _, m := newEnv("-c", "3-", "/f")
	_ = m.WriteFile("/f", []byte("abcdef\n"))
	_ = Run(e)
	if got := out.String(); got != "cdef\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutCharsHeadRange(t *testing.T) {
	e, out, _, m := newEnv("-c", "-2", "/f")
	_ = m.WriteFile("/f", []byte("abcdef\n"))
	_ = Run(e)
	if got := out.String(); got != "ab\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutCharsMultiList(t *testing.T) {
	e, out, _, m := newEnv("-c", "1,3,5", "/f")
	_ = m.WriteFile("/f", []byte("abcdef\n"))
	_ = Run(e)
	if got := out.String(); got != "ace\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutCharsBeyondEnd(t *testing.T) {
	e, out, _, m := newEnv("-c", "10", "/f")
	_ = m.WriteFile("/f", []byte("ab\n"))
	_ = Run(e)
	if got := out.String(); got != "\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutFields(t *testing.T) {
	e, out, _, m := newEnv("-f", "2", "-d", ",", "/f")
	_ = m.WriteFile("/f", []byte("a,b,c\nx,y,z\n"))
	_ = Run(e)
	if got := out.String(); got != "b\ny\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutFieldsRange(t *testing.T) {
	e, out, _, m := newEnv("-f", "2-3", "-d", ",", "/f")
	_ = m.WriteFile("/f", []byte("a,b,c,d\n"))
	_ = Run(e)
	if got := out.String(); got != "b,c\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutFieldsOpenRange(t *testing.T) {
	e, out, _, m := newEnv("-f", "2-", "-d", ",", "/f")
	_ = m.WriteFile("/f", []byte("a,b,c,d\n"))
	_ = Run(e)
	if got := out.String(); got != "b,c,d\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutFieldsBeyond(t *testing.T) {
	e, out, _, m := newEnv("-f", "10", "-d", ",", "/f")
	_ = m.WriteFile("/f", []byte("a,b\n"))
	_ = Run(e)
	if got := out.String(); got != "\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutFieldsNoDelim(t *testing.T) {
	e, out, _, m := newEnv("-f", "1", "-d", ",", "/f")
	_ = m.WriteFile("/f", []byte("nodelim\n"))
	_ = Run(e)
	if got := out.String(); got != "nodelim\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutBytes(t *testing.T) {
	e, out, _, m := newEnv("-b", "1-2", "/f")
	_ = m.WriteFile("/f", []byte("abcdef\n"))
	_ = Run(e)
	if got := out.String(); got != "ab\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"cut", "-c", "1"},
		Stdin:  strings.NewReader("ab\ncd\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "a\nc\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"cut", "-c", "1"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestCutStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"cut", "-c", "1"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestCutMissingFile(t *testing.T) {
	e, _, errb, _ := newEnv("-c", "1", "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCutDashFNoVal(t *testing.T) {
	for _, flag := range []string{"-f", "-c", "-b", "-d"} {
		e, _, errb, _ := newEnv(flag)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", flag, rc)
		}
		if !strings.Contains(errb.String(), "requires an argument") {
			t.Errorf("stderr(%s) = %q", flag, errb.String())
		}
	}
}

// Glued flag forms (`-c1`, `-d,`) match GNU cut.
func TestCutGluedFlags(t *testing.T) {
	e, out, _, m := newEnv("-c1", "/f")
	_ = m.WriteFile("/f", []byte("abc\n"))
	_ = Run(e)
	if got := out.String(); got != "a\n" {
		t.Errorf("got %q", got)
	}
	e2, out2, _, m2 := newEnv("-f1", "-d,", "/f")
	_ = m2.WriteFile("/f", []byte("a,b\n"))
	_ = Run(e2)
	if got := out2.String(); got != "a\n" {
		t.Errorf("got %q", got)
	}
}

func TestCutNoMode(t *testing.T) {
	e, _, errb, _ := newEnv("/f")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is required") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCutBadList(t *testing.T) {
	for _, list := range []string{"", "abc", "0", "1-foo", "foo-1", ",", "1,,2", "1--3", "-0"} {
		e, _, errb, _ := newEnv("-c", list)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%q) = %d", list, rc)
		}
		if !strings.Contains(errb.String(), "invalid list") {
			t.Errorf("stderr(%q) = %q", list, errb.String())
		}
	}
}

func TestCutIsDir(t *testing.T) {
	e, _, errb, m := newEnv("-c", "1", "/d")
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCutPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}

func TestCutMulti(t *testing.T) {
	e, out, _, m := newEnv("-c", "1", "/a", "/b")
	_ = m.WriteFile("/a", []byte("ab\n"))
	_ = m.WriteFile("/b", []byte("cd\n"))
	_ = Run(e)
	if got := out.String(); got != "a\nc\n" {
		t.Errorf("got %q", got)
	}
}
