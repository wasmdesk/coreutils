// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package uname

import (
	"bytes"
	"testing"

	"github.com/wasmdesk/coreutils/cmd/hostname"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"uname"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return out.String(), errb.String(), rc
}

func TestUname(t *testing.T) {
	oldOS, oldArch, oldEnv := GOOS, GOARCH, hostname.LookupEnv
	defer func() { GOOS, GOARCH, hostname.LookupEnv = oldOS, oldArch, oldEnv }()
	hostname.LookupEnv = func(string) (string, bool) { return "h", true }

	GOOS, GOARCH = "wasip1", "wasm"
	cases := map[string]string{
		"":      "WASI\n",
		"-s":    "WASI\n",
		"-a":    "WASI h preview1 wasip1 wasm32 WASI\n",
		"--all": "WASI h preview1 wasip1 wasm32 WASI\n",
		"-nm":   "h wasm32\n",
		"-r":    "preview1\n",
		"-o":    "WASI\n",
		"-pi":   "unknown unknown\n",
		"-v":    "wasip1\n",
	}
	for arg, want := range cases {
		var out string
		var rc int
		if arg == "" {
			out, _, rc = run()
		} else {
			out, _, rc = run(arg)
		}
		if out != want || rc != exit.Ok {
			t.Errorf("uname %s = %q %d, want %q", arg, out, rc, want)
		}
	}
	GOOS = "js"
	if out, _, _ := run("-s"); out != "Browser\n" {
		t.Errorf("js: %q", out)
	}
	GOOS, GOARCH = "darwin", "arm64"
	if out, _, _ := run("-sm"); out != "Darwin arm64\n" {
		t.Errorf("darwin: %q", out)
	}
	GOOS, GOARCH = "linux", "amd64"
	if out, _, _ := run("-sm"); out != "Linux x86_64\n" {
		t.Errorf("linux: %q", out)
	}
	GOARCH = "riscv64"
	if out, _, _ := run("-m"); out != "riscv64\n" {
		t.Errorf("other arch: %q", out)
	}
	if _, _, rc := run("-z"); rc != exit.Usage {
		t.Error("bad option accepted")
	}
	if _, _, rc := run("x"); rc != exit.Usage {
		t.Error("operand accepted")
	}
	if Run(&fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer)}) != exit.Ok {
		t.Error("empty argv")
	}
}
