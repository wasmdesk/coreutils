// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package yes

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"yes"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestYesDefault(t *testing.T) {
	e, out, _ := newEnv()
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "y\n" {
		t.Errorf("got %q", got)
	}
}

func TestYesString(t *testing.T) {
	e, out, _ := newEnv("hello", "world")
	_ = Run(e)
	if got := out.String(); got != "hello world\n" {
		t.Errorf("got %q", got)
	}
}

func TestYesN(t *testing.T) {
	e, out, _ := newEnv("-n", "3")
	_ = Run(e)
	if got := out.String(); got != "y\ny\ny\n" {
		t.Errorf("got %q", got)
	}
}

func TestYesNZero(t *testing.T) {
	e, out, _ := newEnv("-n", "0")
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestYesNCombined(t *testing.T) {
	e, out, _ := newEnv("-n", "2", "ok")
	_ = Run(e)
	if got := out.String(); got != "ok\nok\n" {
		t.Errorf("got %q", got)
	}
}

func TestYesDashNNoValue(t *testing.T) {
	e, _, errb := newEnv("-n")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestYesDashNBad(t *testing.T) {
	for _, v := range []string{"x", "-1"} {
		e, _, errb := newEnv("-n", v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid count") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestYesEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}
