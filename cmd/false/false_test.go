// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package falsecmd

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestFalse(t *testing.T) {
	env := &fsx.Env{Args: []string{"false", "ignored"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(env); rc != exit.Fail {
		t.Errorf("rc = %d, want %d", rc, exit.Fail)
	}
}
