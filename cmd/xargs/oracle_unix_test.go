// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build unix

package xargs

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/fsx"
	"github.com/wasmdesk/coreutils/pkg/procx"
)

// TestXargsOracle runs ours (through pkg/procx: real programs) and the
// system xargs on the same input and compares output and status. Only
// what POSIX defines, so that GNU and BSD agree; -s is left out (each
// counts the environment differently).
func TestXargsOracle(t *testing.T) {
	ref, err := exec.LookPath("xargs")
	if err != nil {
		t.Skip("no system xargs")
	}
	for _, c := range []struct {
		in   string
		args []string
	}{
		{"a b\nc\n", nil},
		{"a b c d e\n", []string{"-n", "2", "echo", "x"}},
		{"'a b' \"c d\" e\\ f\n", []string{"-n1"}},
		{"1 2\n3\n4 5 6\n", []string{"-L", "1"}},
		{"1 2 \n3\n4\n", []string{"-L1"}},
		{"one two\nthree\n", []string{"-I", "{}", "echo", "[{}]", "<{}>"}},
		{"a\x00b c\x00", []string{"-0", "-n1"}},
		{"a b STOP c\n", []string{"-E", "STOP"}},
		{"a b c\n", []string{"-n1", "sh", "-c", "exit 3"}},
		{"a b c\n", []string{"-n1", "sh", "-c", "echo $0; exit 255"}},
		{"", []string{"-r", "echo", "nothing"}},
		{"a b\n", []string{"no-such-program-xyz"}},
	} {
		cmd := exec.Command(ref, c.args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = strings.NewReader(c.in)
		want, err := cmd.Output()
		wantCode := 0
		if ee, ok := err.(*exec.ExitError); ok {
			wantCode = ee.ExitCode()
		}
		var out, errb bytes.Buffer
		cwd, _ := os.Getwd()
		code := Run(&fsx.Env{Args: append([]string{"xargs"}, c.args...), Stdin: strings.NewReader(c.in), Stdout: &out, Stderr: &errb, Cwd: cwd, FS: fsx.NewOSFS(), Exec: procx.Default()})
		// POSIX gives 1-125 for a failed invocation without fixing the
		// value: GNU says 123 or 124 (as we do), BSD says 1.
		failed := func(c int) bool { return c >= 1 && c <= 125 }
		if out.String() != string(want) || (code != wantCode && !(failed(code) && failed(wantCode))) {
			t.Errorf("xargs %q <<< %q:\n got %q rc %d %s\nwant %q rc %d", c.args, c.in, out.String(), code, errb.String(), want, wantCode)
		}
	}
}
