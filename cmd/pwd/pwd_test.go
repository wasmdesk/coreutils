// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package pwd

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestPwd(t *testing.T) {
	var out bytes.Buffer
	env := &fsx.Env{Stdout: &out, Cwd: "/some/where", Args: []string{"pwd"}}
	if rc := Run(env); rc != exit.Ok {
		t.Fatalf("rc = %d, want Ok", rc)
	}
	if got, want := out.String(), "/some/where\n"; got != want {
		t.Errorf("out = %q, want %q", got, want)
	}
}
