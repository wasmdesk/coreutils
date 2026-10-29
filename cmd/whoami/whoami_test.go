// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package whoami

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestWhoami(t *testing.T) {
	var out bytes.Buffer
	if rc := Run(&fsx.Env{Args: []string{"whoami"}, Stdout: &out, Stderr: new(bytes.Buffer)}); rc != 0 || out.String() != "root\n" {
		t.Errorf("%q %d", out.String(), rc)
	}
}
