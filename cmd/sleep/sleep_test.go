// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sleep

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Replace the real sleep with a recorder so we can assert the requested
// duration without actually blocking.
func swap(t *testing.T) *time.Duration {
	t.Helper()
	prev := SleepFn
	rec := new(time.Duration)
	SleepFn = func(d time.Duration) { *rec = d }
	t.Cleanup(func() { SleepFn = prev })
	return rec
}

func newEnv(args ...string) (*fsx.Env, *bytes.Buffer) {
	var errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"sleep"}, args...), Stdout: new(bytes.Buffer), Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"}, &errb
}

func TestSleepInt(t *testing.T) {
	rec := swap(t)
	e, _ := newEnv("2")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if *rec != 2*time.Second {
		t.Errorf("slept %v, want 2s", *rec)
	}
}

func TestSleepFloat(t *testing.T) {
	rec := swap(t)
	e, _ := newEnv("0.5")
	_ = Run(e)
	if *rec != 500*time.Millisecond {
		t.Errorf("slept %v, want 500ms", *rec)
	}
}

func TestSleepMultiple(t *testing.T) {
	rec := swap(t)
	e, _ := newEnv("1", "2", "3")
	_ = Run(e)
	if *rec != 6*time.Second {
		t.Errorf("slept %v, want 6s", *rec)
	}
}

func TestSleepNoArgs(t *testing.T) {
	swap(t)
	e, errb := newEnv()
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "missing operand") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestSleepBad(t *testing.T) {
	swap(t)
	for _, v := range []string{"foo", "-1"} {
		e, errb := newEnv(v)
		if rc := Run(e); rc != exit.Usage {
			t.Errorf("rc(%s) = %d", v, rc)
		}
		if !strings.Contains(errb.String(), "invalid time interval") {
			t.Errorf("stderr(%s) = %q", v, errb.String())
		}
	}
}

func TestSleepEmptyArgv(t *testing.T) {
	swap(t)
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}
