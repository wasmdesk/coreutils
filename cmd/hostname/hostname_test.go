// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package hostname

import (
	"bytes"
	"errors"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"hostname"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return out.String(), errb.String(), rc
}

func TestHostname(t *testing.T) {
	oldEnv, oldHost := LookupEnv, Hostname
	defer func() { LookupEnv, Hostname = oldEnv, oldHost }()

	LookupEnv = func(string) (string, bool) { return "c0ffee123456", true }
	if out, _, rc := run(); out != "c0ffee123456\n" || rc != exit.Ok {
		t.Errorf("from $HOSTNAME: %q %d", out, rc)
	}
	LookupEnv = func(string) (string, bool) { return "", false }
	Hostname = func() (string, error) { return "box", nil }
	if out, _, _ := run(); out != "box\n" {
		t.Errorf("from the OS: %q", out)
	}
	Hostname = func() (string, error) { return "", errors.New("no") }
	if out, _, _ := run(); out != "localhost\n" {
		t.Errorf("fallback: %q", out)
	}
	if _, errs, rc := run("newname"); rc != exit.Usage || errs == "" {
		t.Errorf("set: %d %q", rc, errs)
	}
	if out, _, rc := run(); Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}) != exit.Ok || rc != exit.Ok || out == "" {
		t.Error("empty argv")
	}
}
