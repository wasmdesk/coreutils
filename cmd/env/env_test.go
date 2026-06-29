// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package envcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func stub(t *testing.T, env []string) {
	t.Helper()
	prev := Source
	Source = func() []string { return env }
	t.Cleanup(func() { Source = prev })
}

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"env"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestEnvSorted(t *testing.T) {
	stub(t, []string{"FOO=1", "BAR=2", "BAZ=3"})
	e, out, _ := newEnv()
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "BAR=2\nBAZ=3\nFOO=1\n" {
		t.Errorf("got %q", got)
	}
}

func TestEnvEmpty(t *testing.T) {
	stub(t, nil)
	e, out, _ := newEnv()
	_ = Run(e)
	if got := out.String(); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestEnvArgsRejected(t *testing.T) {
	stub(t, nil)
	e, _, errb := newEnv("X=1")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "not supported") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestEnvEmptyArgv(t *testing.T) {
	stub(t, nil)
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}
