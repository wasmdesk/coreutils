// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package multicall

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestDispatchKnown(t *testing.T) {
	var out bytes.Buffer
	env := &fsx.Env{
		Args:   []string{"pwd"},
		Stdout: &out,
		Stderr: new(bytes.Buffer),
		Cwd:    "/home",
		FS:     fsx.NewMemFS(),
	}
	if rc := Dispatch("pwd", env); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "/home\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}

func TestDispatchUnknown(t *testing.T) {
	var errb bytes.Buffer
	env := &fsx.Env{
		Args:   []string{"frob"},
		Stdout: new(bytes.Buffer),
		Stderr: &errb,
		Cwd:    "/",
		FS:     fsx.NewMemFS(),
	}
	if rc := Dispatch("frob", env); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "command not found") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestHas(t *testing.T) {
	if !Has("pwd") {
		t.Errorf("Has(pwd) = false")
	}
	if Has("frob") {
		t.Errorf("Has(frob) = true")
	}
}

func TestNames(t *testing.T) {
	got := Names()
	if len(got) != 15 {
		t.Errorf("Names len = %d, want 15", len(got))
	}
	// Verify sorted + presence of a few known names.
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Errorf("Names not sorted: %v", got)
		}
	}
	contains := func(s string) bool {
		for _, n := range got {
			if n == s {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"cat", "ls", "pwd", "wc", "find"} {
		if !contains(want) {
			t.Errorf("Names missing %q: %v", want, got)
		}
	}
}

// Smoke-test that every registered tool is invocable without panicking on a
// minimal env (most will return Usage because of missing operand; that's
// fine -- we're proving the dispatch wiring + Run signature consistency).
func TestDispatchAllSmoke(t *testing.T) {
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			env := &fsx.Env{
				Args:   []string{name},
				Stdout: new(bytes.Buffer),
				Stderr: new(bytes.Buffer),
				Cwd:    "/",
				FS:     fsx.NewMemFS(),
			}
			// Any non-negative return is fine; we only care that the dispatch
			// did not panic and the registry entry actually points somewhere.
			_ = Dispatch(name, env)
		})
	}
}
