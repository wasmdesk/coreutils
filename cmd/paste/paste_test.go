// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package paste

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
	return &fsx.Env{Args: append([]string{"paste"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestPasteTwo(t *testing.T) {
	e, out, _, m := newEnv("/a", "/b")
	_ = m.WriteFile("/a", []byte("1\n2\n3\n"))
	_ = m.WriteFile("/b", []byte("x\ny\nz\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "1\tx\n2\ty\n3\tz\n" {
		t.Errorf("got %q", got)
	}
}

func TestPasteDelim(t *testing.T) {
	e, out, _, m := newEnv("-d", ",", "/a", "/b")
	_ = m.WriteFile("/a", []byte("1\n2\n"))
	_ = m.WriteFile("/b", []byte("x\ny\n"))
	_ = Run(e)
	if got := out.String(); got != "1,x\n2,y\n" {
		t.Errorf("got %q", got)
	}
}

func TestPasteUneven(t *testing.T) {
	e, out, _, m := newEnv("/a", "/b")
	_ = m.WriteFile("/a", []byte("1\n2\n3\n"))
	_ = m.WriteFile("/b", []byte("x\n"))
	_ = Run(e)
	if got := out.String(); got != "1\tx\n2\t\n3\t\n" {
		t.Errorf("got %q", got)
	}
}

func TestPasteSelf(t *testing.T) {
	e, out, _, m := newEnv("/a", "/a")
	_ = m.WriteFile("/a", []byte("1\n2\n"))
	_ = Run(e)
	if got := out.String(); got != "1\t1\n2\t2\n" {
		t.Errorf("got %q", got)
	}
}

func TestPasteMissingFile(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPasteIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPasteNoArgs(t *testing.T) {
	e, _, errb, _ := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPasteDashDNoVal(t *testing.T) {
	e, _, errb, _ := newEnv("-d")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPasteEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestPastePrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}

func TestPasteEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
