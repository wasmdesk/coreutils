// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package expr

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"expr"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestExprArith(t *testing.T) {
	cases := []struct {
		a, op, b string
		out      string
		rc       int
	}{
		{"2", "+", "3", "5\n", exit.Ok},
		{"10", "-", "4", "6\n", exit.Ok},
		{"3", "*", "4", "12\n", exit.Ok},
		{"10", "/", "3", "3\n", exit.Ok},
		{"10", "%", "3", "1\n", exit.Ok},
		{"3", "-", "3", "0\n", exit.Fail}, // zero-result -> Fail
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.a, c.op, c.b)
		if rc := Run(e); rc != c.rc {
			t.Errorf("%s %s %s: rc = %d, want %d", c.a, c.op, c.b, rc, c.rc)
		}
		if got := out.String(); got != c.out {
			t.Errorf("%s %s %s: out = %q, want %q", c.a, c.op, c.b, got, c.out)
		}
	}
}

func TestExprCompareNumeric(t *testing.T) {
	cases := []struct {
		a, op, b string
		out      string
		rc       int
	}{
		{"2", "=", "2", "1\n", exit.Ok},
		{"2", "!=", "3", "1\n", exit.Ok},
		{"2", "<", "3", "1\n", exit.Ok},
		{"3", "<=", "3", "1\n", exit.Ok},
		{"4", ">", "3", "1\n", exit.Ok},
		{"4", ">=", "4", "1\n", exit.Ok},
		{"2", "=", "3", "0\n", exit.Fail},
		{"3", "!=", "3", "0\n", exit.Fail},
		{"3", "<", "2", "0\n", exit.Fail},
		{"4", "<=", "3", "0\n", exit.Fail},
		{"2", ">", "3", "0\n", exit.Fail},
		{"3", ">=", "4", "0\n", exit.Fail},
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.a, c.op, c.b)
		if rc := Run(e); rc != c.rc {
			t.Errorf("%s %s %s: rc = %d, want %d", c.a, c.op, c.b, rc, c.rc)
		}
		if got := out.String(); got != c.out {
			t.Errorf("%s %s %s: out = %q, want %q", c.a, c.op, c.b, got, c.out)
		}
	}
}

func TestExprCompareLex(t *testing.T) {
	cases := []struct {
		a, op, b string
		out      string
		rc       int
	}{
		{"foo", "=", "foo", "1\n", exit.Ok},
		{"foo", "!=", "bar", "1\n", exit.Ok},
		{"abc", "<", "abd", "1\n", exit.Ok},
		{"abc", "<=", "abc", "1\n", exit.Ok},
		{"abd", ">", "abc", "1\n", exit.Ok},
		{"abc", ">=", "abc", "1\n", exit.Ok},
		{"foo", "=", "bar", "0\n", exit.Fail},
		{"foo", "!=", "foo", "0\n", exit.Fail},
		{"abd", "<", "abc", "0\n", exit.Fail},
		{"abd", "<=", "abc", "0\n", exit.Fail},
		{"abc", ">", "abd", "0\n", exit.Fail},
		{"abc", ">=", "abd", "0\n", exit.Fail},
	}
	for _, c := range cases {
		e, out, _ := newEnv(c.a, c.op, c.b)
		if rc := Run(e); rc != c.rc {
			t.Errorf("%s %s %s: rc = %d", c.a, c.op, c.b, rc)
		}
		if got := out.String(); got != c.out {
			t.Errorf("%s %s %s: out = %q", c.a, c.op, c.b, got)
		}
	}
}

func TestExprRegexMatch(t *testing.T) {
	e, out, _ := newEnv("hello", ":", "he..o")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "5\n" {
		t.Errorf("got %q", got)
	}
}

func TestExprRegexNoMatch(t *testing.T) {
	e, out, _ := newEnv("hello", ":", "world")
	if rc := Run(e); rc != exit.Fail {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "0\n" {
		t.Errorf("got %q", got)
	}
}

func TestExprRegexBad(t *testing.T) {
	e, _, errb := newEnv("hello", ":", "[unterm")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid regex") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExprDivByZero(t *testing.T) {
	for _, op := range []string{"/", "%"} {
		e, _, errb := newEnv("5", op, "0")
		if rc := Run(e); rc != exit.Fail {
			t.Errorf("rc(%s) = %d", op, rc)
		}
		if !strings.Contains(errb.String(), "division by zero") {
			t.Errorf("stderr(%s) = %q", op, errb.String())
		}
	}
}

func TestExprNonInt(t *testing.T) {
	e, _, errb := newEnv("foo", "+", "1")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "non-integer") {
		t.Errorf("stderr = %q", errb.String())
	}
	e2, _, errb2 := newEnv("1", "+", "bar")
	if rc := Run(e2); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb2.String(), "non-integer") {
		t.Errorf("stderr2 = %q", errb2.String())
	}
}

func TestExprUnknownOp(t *testing.T) {
	e, _, errb := newEnv("1", "@", "2")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "unknown operator") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExprBadArity(t *testing.T) {
	e, _, errb := newEnv("1", "+")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "usage") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestExprArithUnreachable(t *testing.T) {
	// Direct call to drive the defensive default branch in arith.
	if _, err := arith(1, 2, "bogus"); err == nil {
		t.Errorf("arith(bogus) returned nil err")
	}
}

func TestExprEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
