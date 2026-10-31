// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build !wasip1 && !js

package procx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// The test binary is its own child: PROCX_HELPER selects what it does.
func TestMain(m *testing.M) {
	switch os.Getenv("PROCX_HELPER") {
	case "":
		os.Exit(m.Run())
	case "echo":
		fmt.Println(strings.Join(os.Args[1:], " "))
		os.Exit(0)
	case "cat":
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	case "pwd-path":
		wd, _ := os.Getwd()
		fmt.Println(filepath.Base(wd), os.Getenv("PROBE"))
		os.Exit(0)
	case "exit":
		n, _ := strconv.Atoi(os.Args[1])
		os.Exit(n)
	}
}

func self(t *testing.T) (dir, name string) {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(p), filepath.Base(p)
}

func helperEnv(mode string, extra ...string) []string {
	return append(append(os.Environ(), "PROCX_HELPER="+mode), extra...)
}

func TestRun(t *testing.T) {
	dir, name := self(t)
	run := Default()
	// Found on the PATH of the command's own environment.
	var out bytes.Buffer
	code, err := run(fsx.Command{Name: name, Args: []string{name, "a", "b"}, Env: helperEnv("echo", "PATH="+dir), Stdout: &out})
	if err != nil || code != 0 || out.String() != "a b\n" {
		t.Fatalf("%d %v %q", code, err, out.String())
	}
	// Stdin, and an exit status.
	out.Reset()
	if code, err := run(fsx.Command{Name: name, Args: []string{name}, Env: helperEnv("cat", "PATH="+dir), Stdin: strings.NewReader("in"), Stdout: &out}); err != nil || code != 0 || out.String() != "in" {
		t.Errorf("cat: %d %v %q", code, err, out.String())
	}
	if code, err := run(fsx.Command{Name: name, Args: []string{name, "7"}, Env: helperEnv("exit", "PATH="+dir)}); err != nil || code != 7 {
		t.Errorf("exit: %d %v", code, err)
	}
	// Dir and Env reach the child; a path with a slash, relative to Dir.
	out.Reset()
	sub := t.TempDir()
	rel, _ := filepath.Rel(sub, filepath.Join(dir, name))
	if code, err := run(fsx.Command{Name: rel, Args: []string{name}, Dir: sub, Env: helperEnv("pwd-path", "PROBE=yes"), Stdout: &out}); err != nil || code != 0 || out.String() != filepath.Base(sub)+" yes\n" {
		t.Errorf("dir: %d %v %q", code, err, out.String())
	}
	// Not found: on PATH, through an empty or relative element, and by path.
	for _, c := range []fsx.Command{
		{Name: "no-such-program-xyz", Env: []string{"PATH=" + dir}},
		{Name: "no-such-program-xyz", Env: []string{"PATH=:rel"}, Dir: dir},
		{Name: "./no-such-program-xyz", Dir: dir},
	} {
		if _, err := run(c); !errors.Is(err, fsx.ErrNoProgram) {
			t.Errorf("%+v: %v", c, err)
		}
	}
	// Found, but cannot start: a working directory that does not exist.
	if _, err := run(fsx.Command{Name: name, Args: []string{name}, Env: helperEnv("echo", "PATH="+dir), Dir: filepath.Join(sub, "gone")}); err == nil || errors.Is(err, fsx.ErrNoProgram) {
		t.Errorf("bad dir: %v", err)
	}
	// It ran, but its output could not be copied: the status still counts.
	if code, err := run(fsx.Command{Name: name, Args: []string{name, "x"}, Env: helperEnv("echo", "PATH="+dir), Stdout: failWriter{}}); err != nil || code != 0 {
		t.Errorf("copy error: %d %v", code, err)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestPathOf(t *testing.T) {
	t.Setenv("PATH", "/from/parent")
	if p := pathOf(nil); p != "/from/parent" {
		t.Errorf("nil env: %q", p)
	}
	if p := pathOf([]string{"PATH=/a", "X=1", "PATH=/b"}); p != "/b" {
		t.Errorf("last wins: %q", p)
	}
	if p := pathOf([]string{"X=1"}); p != "" {
		t.Errorf("no PATH: %q", p)
	}
}

func TestStatus(t *testing.T) {
	if status(-1) != 137 || status(3) != 3 {
		t.Error(status(-1), status(3))
	}
}
