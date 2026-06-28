// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package echo

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func TestEcho(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"single", []string{"echo", "hello"}, "hello\n"},
		{"multi", []string{"echo", "a", "b", "c"}, "a b c\n"},
		{"bare", []string{"echo"}, "\n"},
		{"empty-argv", nil, "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			env := &fsx.Env{Stdout: &out, Args: c.args}
			if rc := Run(env); rc != exit.Ok {
				t.Fatalf("rc = %d, want Ok", rc)
			}
			if got := out.String(); got != c.want {
				t.Errorf("out = %q, want %q", got, c.want)
			}
		})
	}
}
