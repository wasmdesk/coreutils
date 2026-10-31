// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package xargs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// fake records each command line as one output line "[a|b|c]", and exits
// with the status the program name asks for ("exitN").
type fake struct{ calls []string }

func (f *fake) exec(c fsx.Command) (int, error) {
	f.calls = append(f.calls, "["+strings.Join(c.Args, "|")+"]")
	if c.Stdin != nil {
		return 99, nil // xargs gives its utility no stdin
	}
	switch {
	case c.Name == "missing":
		return 0, fsx.ErrNoProgram
	case c.Name == "broken":
		return 0, errors.New("ENOEXEC")
	case strings.HasPrefix(c.Name, "exit"):
		var n int
		fmt.Sscanf(c.Name, "exit%d", &n)
		return n, nil
	}
	return 0, nil
}

func run(stdin string, args ...string) (*fake, string, int) {
	f := &fake{}
	var errb bytes.Buffer
	e := &fsx.Env{Args: append([]string{"xargs"}, args...), Stdout: io.Discard, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/w", Exec: f.exec}
	if stdin != "\x00none" {
		e.Stdin = strings.NewReader(stdin)
	}
	rc := Run(e)
	return f, errb.String(), rc
}

func TestXargs(t *testing.T) {
	for _, c := range []struct {
		in   string
		args []string
		want string
		rc   int
	}{
		{"a b\nc\n", nil, "[echo|a|b|c]", 0},
		{"a b c d e\n", []string{"-n", "2", "cmd", "x"}, "[cmd|x|a|b][cmd|x|c|d][cmd|x|e]", 0},
		{"'a b' \"c d\" e\\ f g\\\\h\n", []string{"-n1"}, "[echo|a b][echo|c d][echo|e f][echo|g\\h]", 0},
		{"1 2\n3\n4 5 6\n", []string{"-L", "1"}, "[echo|1|2][echo|3][echo|4|5|6]", 0},
		{"1 2 \n3\n4\n", []string{"-L1"}, "[echo|1|2|3][echo|4]", 0}, // a trailing blank continues the line
		{"1\n2\n3\n", []string{"-L2"}, "[echo|1|2][echo|3]", 0},
		{"  one two\n\nthree\n", []string{"-I", "{}", "cp", "{}", "/d/{}.bak"}, "[cp|one two|/d/one two.bak][cp|three|/d/three.bak]", 0},
		{"a\x00b c\x00", []string{"-0"}, "[echo|a|b c]", 0},
		{"a\x00b", []string{"-0", "-n1"}, "[echo|a][echo|b]", 0},
		{"a b STOP c\n", []string{"-E", "STOP"}, "[echo|a|b]", 0},
		{"a b\n", []string{"-E", ""}, "[echo|a|b]", 0},
		{"aaaa bbbb cccc\n", []string{"-s", "15"}, "[echo|aaaa|bbbb][echo|cccc]", 0}, // echo(5)+5+5=15
		{"", nil, "[echo]", 0},
		{"\n \n", []string{"-r"}, "", 0},
		{"\x00none", nil, "[echo]", 0},
		{"a\n", []string{"--", "-x"}, "[-x|a]", 0},
		{"a b\n", []string{"-t", "-n1"}, "[echo|a][echo|b]", 0},
		// Statuses: 123 goes on, 255 stops (124), 126/127 stop.
		{"a b\n", []string{"-n1", "exit3"}, "[exit3|a][exit3|b]", StatusFailed},
		{"a b\n", []string{"-n1", "exit255"}, "[exit255|a]", StatusAborted},
		{"a b\n", []string{"-n1", "missing"}, "[missing|a]", 127},
		{"a b\n", []string{"-n1", "broken"}, "[broken|a]", 126},
		{"a b\n", []string{"-I", "X", "exit255", "X"}, "[exit255|a b]", StatusAborted},
	} {
		f, errb, rc := run(c.in, c.args...)
		if got := strings.Join(f.calls, ""); got != c.want || rc != c.rc {
			t.Errorf("xargs %q <<< %q:\n got %s rc %d %s\nwant %s rc %d", c.args, c.in, got, rc, errb, c.want, c.rc)
		}
	}
}

func TestXargsTrace(t *testing.T) {
	_, errb, _ := run("a b\n", "-t", "cmd")
	if errb != "cmd a b\n" {
		t.Errorf("%q", errb)
	}
}

func TestXargsErrors(t *testing.T) {
	for _, c := range []struct {
		in   string
		args []string
		rc   int
		msg  string
	}{
		{"", []string{"-z"}, exit.Usage, "invalid option -- 'z'"},
		{"", []string{"-n"}, exit.Usage, "requires an argument -- 'n'"},
		{"", []string{"-n", "0"}, exit.Usage, "invalid number"},
		{"", []string{"-sx"}, exit.Usage, "invalid number"},
		{"", []string{"-p"}, exit.Usage, "-p is not supported"},
		{"", []string{"-I", ""}, exit.Usage, "replacement string is empty"},
		{"'a\n", nil, 1, "unmatched quote"},
		{"'a", nil, 1, "unmatched quote"},
		{"a\\", nil, 1, "backslash at end of input"},
		{"aaaaaaaaaa\n", []string{"-s", "8"}, 1, "argument line too long"},
		{"a bbbbbbbbbb\n", []string{"-x", "-s", "9"}, 1, "argument line too long"},
		{"x\n", []string{"-s", "3", "long-utility"}, 1, "do not fit"},
		{"x\n", []string{"-s", "5", "-I", "{}", "{}{}{}{}{}"}, 1, "argument line too long"},
		{strings.Repeat("y", MaxArg+2), nil, 1, "argument line too long"},
		{strings.Repeat("y", MaxArg+2), []string{"-0"}, 1, "argument line too long"},
	} {
		_, errb, rc := run(c.in, c.args...)
		if rc != c.rc || !strings.Contains(errb, c.msg) {
			t.Errorf("xargs %q <<< %.20q: rc %d %q, want %d %q", c.args, c.in, rc, errb, c.rc, c.msg)
		}
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("EIO") }

func TestXargsReadError(t *testing.T) {
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"xargs", "-0"}, Stdin: failReader{}, Stdout: io.Discard, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/", Exec: (&fake{}).exec}
	if rc := Run(e); rc != 1 || !strings.Contains(errb.String(), "EIO") {
		t.Errorf("rc %d %q", rc, errb.String())
	}
}
