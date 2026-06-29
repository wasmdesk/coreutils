// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package seq

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"seq"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestSeqOneArg(t *testing.T) {
	e, out, _ := newEnv("5")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "1\n2\n3\n4\n5\n" {
		t.Errorf("got %q", got)
	}
}

func TestSeqTwoArgs(t *testing.T) {
	e, out, _ := newEnv("3", "5")
	_ = Run(e)
	if got := out.String(); got != "3\n4\n5\n" {
		t.Errorf("got %q", got)
	}
}

func TestSeqThreeArgs(t *testing.T) {
	e, out, _ := newEnv("1", "2", "9")
	_ = Run(e)
	if got := out.String(); got != "1\n3\n5\n7\n9\n" {
		t.Errorf("got %q", got)
	}
}

func TestSeqNegativeStep(t *testing.T) {
	e, out, _ := newEnv("5", "-1", "1")
	_ = Run(e)
	if got := out.String(); got != "5\n4\n3\n2\n1\n" {
		t.Errorf("got %q", got)
	}
}

// Empty sequence when first > last with positive step.
func TestSeqEmpty(t *testing.T) {
	e, out, _ := newEnv("5", "3")
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestSeqNoArgs(t *testing.T) {
	e, _, errb := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "usage") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSeqTooManyArgs(t *testing.T) {
	e, _, errb := newEnv("1", "2", "3", "4")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "usage") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSeqBadInt(t *testing.T) {
	e, _, errb := newEnv("foo")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid integer") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSeqZeroStep(t *testing.T) {
	e, _, errb := newEnv("1", "0", "5")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "Zero increment") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSeqEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
