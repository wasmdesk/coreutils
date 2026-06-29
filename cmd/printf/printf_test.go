// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package printf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"printf"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestPrintfBasic(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"hello\\n"}, "hello\n"},
		{[]string{"%s=%s\\n", "a", "b"}, "a=b\n"},
		{[]string{"%d\\n", "42"}, "42\n"},
		{[]string{"%x\\n", "255"}, "ff\n"},
		{[]string{"%o\\n", "8"}, "10\n"},
		{[]string{"%c\\n", "A"}, "A\n"},
		{[]string{"100%%\\n"}, "100%\n"},
		{[]string{"%s\\t%s\\n", "x", "y"}, "x\ty\n"},
		{[]string{"\\\\\\n"}, "\\\n"},
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.args...)
		if rc := Run(e); rc != exit.Ok {
			t.Errorf("rc(%v) = %d", c.args, rc)
		}
		if got := out.String(); got != c.want {
			t.Errorf("printf %v = %q, want %q", c.args, got, c.want)
		}
	}
}

// FORMAT reused when there are more operands than specs (GNU semantics).
func TestPrintfReuseFormat(t *testing.T) {
	e, out, _ := newEnv("%s\\n", "a", "b", "c")
	_ = Run(e)
	if got := out.String(); got != "a\nb\nc\n" {
		t.Errorf("got %q", got)
	}
}

// Too few operands -> remaining specs see zero defaults.
func TestPrintfTooFewOperands(t *testing.T) {
	e, out, _ := newEnv("%s|%d\\n", "hi")
	_ = Run(e)
	if got := out.String(); got != "hi|0\n" {
		t.Errorf("got %q", got)
	}
}

func TestPrintfNoArgs(t *testing.T) {
	e, _, errb := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing format") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPrintfBadInt(t *testing.T) {
	e, _, errb := newEnv("%d\\n", "notanumber")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid number") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPrintfBadHex(t *testing.T) {
	e, _, errb := newEnv("%x\\n", "x")
	_ = Run(e)
	if !strings.Contains(errb.String(), "invalid number") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPrintfBadOct(t *testing.T) {
	e, _, errb := newEnv("%o\\n", "x")
	_ = Run(e)
	if !strings.Contains(errb.String(), "invalid number") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPrintfUnsupportedConv(t *testing.T) {
	e, _, errb := newEnv("%q\\n", "x")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "unsupported conversion") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestPrintfTrailingPercent(t *testing.T) {
	e, out, _ := newEnv("done%")
	_ = Run(e)
	if got := out.String(); got != "done%" {
		t.Errorf("got %q", got)
	}
}

func TestPrintfEmptyC(t *testing.T) {
	// %c with empty operand prints nothing.
	e, out, _ := newEnv("[%c]", "")
	_ = Run(e)
	if got := out.String(); got != "[]" {
		t.Errorf("got %q", got)
	}
}

func TestPrintfTrailingPercentInRender(t *testing.T) {
	// Drive the render "trailing %" branch via a 1-arg %d format.
	e, _, _ := newEnv("x%")
	_ = Run(e)
}

func TestPrintfEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
