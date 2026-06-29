// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fold

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
	return &fsx.Env{Args: append([]string{"fold"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestFoldDefault(t *testing.T) {
	// 100-char line, default width 80 -> two lines.
	long := strings.Repeat("a", 100)
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte(long+"\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	want := strings.Repeat("a", 80) + "\n" + strings.Repeat("a", 20) + "\n"
	if got != want {
		t.Errorf("got %q", got)
	}
}

func TestFoldDashW(t *testing.T) {
	e, out, _, m := newEnv("-w", "3", "/f")
	_ = m.WriteFile("/f", []byte("abcdefg\n"))
	_ = Run(e)
	if got := out.String(); got != "abc\ndef\ng\n" {
		t.Errorf("got %q", got)
	}
}

func TestFoldShort(t *testing.T) {
	e, out, _, m := newEnv("-w", "100", "/f")
	_ = m.WriteFile("/f", []byte("hi\n"))
	_ = Run(e)
	if got := out.String(); got != "hi\n" {
		t.Errorf("got %q", got)
	}
}

func TestFoldStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"fold", "-w", "2"},
		Stdin:  strings.NewReader("abcd\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "ab\ncd\n" {
		t.Errorf("got %q", got)
	}
}

func TestFoldStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"fold"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestFoldStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"fold"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestFoldDashWNoVal(t *testing.T) {
	e, _, errb, _ := newEnv("-w")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestFoldDashWBad(t *testing.T) {
	for _, v := range []string{"foo", "0", "-1"} {
		e, _, errb, _ := newEnv("-w", v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid width") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestFoldMissing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestFoldIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestFoldEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestFoldPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
