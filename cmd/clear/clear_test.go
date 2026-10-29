// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package clear

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(args ...string) (string, int) {
	var out bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"clear"}, args...), Stdout: &out, Stderr: new(bytes.Buffer)})
	return out.String(), rc
}

func TestClear(t *testing.T) {
	if out, rc := run(); out != "\x1b[H\x1b[2J\x1b[3J" || rc != exit.Ok {
		t.Errorf("%q %d", out, rc)
	}
	if out, _ := run("-x"); out != "\x1b[H\x1b[2J" {
		t.Errorf("-x: %q", out)
	}
	if _, rc := run("-y"); rc != exit.Usage {
		t.Error("bad option accepted")
	}
	if Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}) != exit.Ok {
		t.Error("empty argv")
	}
}
