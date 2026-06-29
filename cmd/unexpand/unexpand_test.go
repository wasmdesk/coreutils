// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package unexpand

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
	return &fsx.Env{Args: append([]string{"unexpand"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestUnexpandDefault(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("        x\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "\tx\n" {
		t.Errorf("got %q", got)
	}
}

func TestUnexpandPartial(t *testing.T) {
	e, out, _, m := newEnv("-t", "4", "/f")
	_ = m.WriteFile("/f", []byte("      x\n"))
	_ = Run(e)
	// 6 spaces / 4 = 1 tab + 2 spaces.
	if got := out.String(); got != "\t  x\n" {
		t.Errorf("got %q", got)
	}
}

func TestUnexpandNoLeading(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", []byte("nospaces\n"))
	_ = Run(e)
	if got := out.String(); got != "nospaces\n" {
		t.Errorf("got %q", got)
	}
}

func TestUnexpandStdin(t *testing.T) {
	var out bytes.Buffer
	e := &fsx.Env{
		Args:   []string{"unexpand", "-t", "2"},
		Stdin:  strings.NewReader("    x\n"),
		Stdout: &out, Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/",
	}
	_ = Run(e)
	if got := out.String(); got != "\t\tx\n" {
		t.Errorf("got %q", got)
	}
}

func TestUnexpandStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"unexpand"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestUnexpandStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"unexpand"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestUnexpandDashTNoVal(t *testing.T) {
	e, _, errb, _ := newEnv("-t")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUnexpandDashTBad(t *testing.T) {
	for _, v := range []string{"foo", "0", "-1"} {
		e, _, errb, _ := newEnv("-t", v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid tabstop") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestUnexpandMissing(t *testing.T) {
	e, _, errb, _ := newEnv("/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUnexpandIsDir(t *testing.T) {
	e, _, errb, m := newEnv("/d")
	_ = m.Mkdir("/d")
	_ = Run(e)
	if !strings.Contains(errb.String(), "is a directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestUnexpandEmpty(t *testing.T) {
	e, out, _, m := newEnv("/f")
	_ = m.WriteFile("/f", nil)
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestUnexpandPrettyErr(t *testing.T) {
	if got := prettyErr(errors.New("boom")); got != "boom" {
		t.Errorf("got %q", got)
	}
}
