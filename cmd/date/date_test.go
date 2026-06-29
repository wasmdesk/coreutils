// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package date

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func freeze(t *testing.T, when time.Time) {
	t.Helper()
	prev := Now
	Now = func() time.Time { return when }
	t.Cleanup(func() { Now = prev })
}

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"date"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &out, &errb
}

func TestDateNow(t *testing.T) {
	freeze(t, time.Date(2026, 6, 29, 12, 34, 56, 0, time.UTC))
	e, out, _ := newEnv()
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := strings.TrimRight(out.String(), "\n")
	if got != "Mon, 29 Jun 2026 12:34:56 UTC" {
		t.Errorf("got %q", got)
	}
}

func TestDateDashD(t *testing.T) {
	freeze(t, time.Now())
	e, out, _ := newEnv("-d", "2020-01-02T03:04:05Z")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := strings.TrimRight(out.String(), "\n")
	if got != "Thu, 02 Jan 2020 03:04:05 UTC" {
		t.Errorf("got %q", got)
	}
}

func TestDateDashDNoVal(t *testing.T) {
	e, _, errb := newEnv("-d")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "requires an argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestDateDashDBad(t *testing.T) {
	e, _, errb := newEnv("-d", "garbage")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "invalid date") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestDateUnknownArg(t *testing.T) {
	e, _, errb := newEnv("--bogus")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "unknown argument") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestDateEmptyArgv(t *testing.T) {
	freeze(t, time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC))
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}
