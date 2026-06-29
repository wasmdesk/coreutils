// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package truecmd

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestTrue(t *testing.T) {
	env := &fsx.Env{Args: []string{"true", "ignored", "args"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(env); rc != exit.Ok {
		t.Errorf("rc = %d, want 0", rc)
	}
}
