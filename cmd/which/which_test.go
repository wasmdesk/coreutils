// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package which

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func setup(t *testing.T) *fsx.MemFS {
	m := fsx.NewMemFS()
	m.MkdirAll("/bin")
	m.MkdirAll("/usr/bin")
	m.MkdirAll("/home/u")
	m.WriteFile("/bin/sh", []byte("x"))
	m.WriteFile("/usr/bin/sh", []byte("x"))
	m.WriteFile("/usr/bin/tool.wasm", []byte("x"))
	m.MkdirAll("/bin/dir")
	m.WriteFile("/home/u/local", []byte("x"))
	old := LookupEnv
	LookupEnv = func(string) (string, bool) { return "/bin:/usr/bin", true }
	t.Cleanup(func() { LookupEnv = old })
	return m
}

func run(m *fsx.MemFS, cwd string, args ...string) (string, int) {
	var out bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"which"}, args...), Stdout: &out, Stderr: new(bytes.Buffer), FS: m, Cwd: cwd})
	return out.String(), rc
}

func TestWhich(t *testing.T) {
	m := setup(t)
	cases := []struct {
		cwd  string
		args []string
		out  string
		rc   int
	}{
		{"/", []string{"sh"}, "/bin/sh\n", exit.Ok},
		{"/", []string{"-a", "sh"}, "/bin/sh\n/usr/bin/sh\n", exit.Ok},
		{"/", []string{"tool"}, "/usr/bin/tool.wasm\n", exit.Ok},
		{"/", []string{"tool.wasm"}, "/usr/bin/tool.wasm\n", exit.Ok},
		{"/", []string{"dir"}, "", exit.Fail},
		{"/", []string{"nope", "sh"}, "/bin/sh\n", exit.Fail},
		{"/home/u", []string{"./local"}, "/home/u/local\n", exit.Ok},
		{"/", []string{"/usr/bin/tool"}, "/usr/bin/tool.wasm\n", exit.Ok},
		{"/", []string{"/usr/bin/none"}, "", exit.Fail},
		{"/", nil, "", exit.Usage},
		{"/", []string{"-x"}, "", exit.Usage},
	}
	for _, c := range cases {
		if out, rc := run(m, c.cwd, c.args...); out != c.out || rc != c.rc {
			t.Errorf("which %v = %q %d, want %q %d", c.args, out, rc, c.out, c.rc)
		}
	}
	// An empty PATH element is the current directory.
	LookupEnv = func(string) (string, bool) { return ":/bin", true }
	if out, _ := run(m, "/home/u", "local"); out != "/home/u/local\n" {
		t.Errorf("empty element: %q", out)
	}
	if Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: m, Cwd: "/"}) != exit.Usage {
		t.Error("empty argv")
	}
}
