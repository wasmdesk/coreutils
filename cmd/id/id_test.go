// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package id

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(fn func(*fsx.Env) int, name string, args ...string) (string, int) {
	var out, errb bytes.Buffer
	rc := fn(&fsx.Env{Args: append([]string{name}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return out.String(), rc
}

func TestID(t *testing.T) {
	cases := []struct {
		args []string
		out  string
		rc   int
	}{
		{nil, "uid=0(root) gid=0(root) groups=0(root)\n", exit.Ok},
		{[]string{"root"}, "uid=0(root) gid=0(root) groups=0(root)\n", exit.Ok},
		{[]string{"-u"}, "0\n", exit.Ok},
		{[]string{"-un"}, "root\n", exit.Ok},
		{[]string{"-g", "-n"}, "root\n", exit.Ok},
		{[]string{"-G"}, "0\n", exit.Ok},
		{[]string{"-ur"}, "0\n", exit.Ok},
		{[]string{"-u", "-g"}, "", exit.Usage},
		{[]string{"-u", "-G"}, "", exit.Usage},
		{[]string{"-g", "-G"}, "", exit.Usage},
		{[]string{"-n"}, "", exit.Usage},
		{[]string{"-x"}, "", exit.Usage},
		{[]string{"alice"}, "", exit.Fail},
	}
	for _, c := range cases {
		if out, rc := run(Run, "id", c.args...); out != c.out || rc != c.rc {
			t.Errorf("id %v = %q %d, want %q %d", c.args, out, rc, c.out, c.rc)
		}
	}
	if Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}) != exit.Ok {
		t.Error("empty argv")
	}
}

func TestWhoami(t *testing.T) {
	if out, rc := run(Whoami, "whoami"); out != "root\n" || rc != exit.Ok {
		t.Errorf("%q %d", out, rc)
	}
	if _, rc := run(Whoami, "whoami", "x"); rc != exit.Usage {
		t.Error("operand accepted")
	}
}
