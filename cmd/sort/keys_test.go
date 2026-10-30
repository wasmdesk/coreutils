// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sort

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(m *fsx.MemFS, stdin string, args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: append([]string{"sort"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}
	if stdin != "" {
		e.Stdin = strings.NewReader(stdin)
	}
	rc := Run(e)
	return out.String(), errb.String(), rc
}

// Expectations that do not need the oracle (fixed by POSIX's text).
func TestSortKeys(t *testing.T) {
	m := fsx.NewMemFS()
	for _, c := range []struct {
		in   string
		args []string
		want string
	}{
		{"b 2\na 10\nc 1\n", []string{"-k", "2n"}, "c 1\nb 2\na 10\n"},
		{"a:3\nb:1\n", []string{"-t:", "-k2"}, "b:1\na:3\n"},
		{"x\n", []string{"-k2"}, "x\n"},                        // a missing field is empty
		{"ab\nac\n", []string{"-k1.5"}, "ab\nac\n"},            // past the end of the line
		{"ba\nab\n", []string{"-k1.2,1.1"}, "ab\nba\n"},        // an empty key: line order
		{"1 b\n1 a\n", []string{"-u", "-k1,1n"}, "1 b\n"},      // -u keeps the first
		{"0.10\n.1\n-.0\n", []string{"-n"}, "-.0\n.1\n0.10\n"}, // equal numbers: by line
		{"9\n10\n", []string{"-n", "--", "-"}, "9\n10\n"},      // -- then stdin as -
		{"b\na\n", []string{"-rn"}, "b\na\n"},
		{"c:b\na\n", []string{"-t:", "-k5"}, "a\nc:b\n"},         // a field past the last: empty
		{"x  ab\ny ba\n", []string{"-k2,2.1b"}, "x  ab\ny ba\n"}, // b on the end position
		{"\x01b\na\n", []string{"-i"}, "a\n\x01b\n"},             // -i drops a control byte
		{"abd\nabc\n", []string{"-k1.3,1.1"}, "abc\nabd\n"},      // end before start: empty key                  // non-numbers are 0, line order reversed
	} {
		out, errb, rc := run(m, c.in, c.args...)
		if out != c.want || rc != exit.Ok {
			t.Errorf("sort %q <<< %q: got %q rc %d %s, want %q", c.args, c.in, out, rc, errb, c.want)
		}
	}
}

func TestSortCheck(t *testing.T) {
	m := fsx.NewMemFS()
	if _, _, rc := run(m, "a\nb\n", "-c"); rc != exit.Ok {
		t.Errorf("sorted: rc %d", rc)
	}
	if _, errb, rc := run(m, "b\na\n", "-c"); rc != exit.Fail || !strings.Contains(errb, "disorder: a") {
		t.Errorf("unsorted: rc %d %q", rc, errb)
	}
	if _, _, rc := run(m, "a\na\n", "-cu"); rc != exit.Fail {
		t.Errorf("-cu with a duplicate: rc %d", rc)
	}
}

func TestSortOutput(t *testing.T) {
	m := fsx.NewMemFS()
	m.WriteFile("/f", []byte("b\na\n"))
	// -o may name an input: it is written after the input is read.
	if out, _, rc := run(m, "", "-o", "/f", "/f"); rc != exit.Ok || out != "" {
		t.Fatalf("rc %d out %q", rc, out)
	}
	if b, _ := m.ReadFile("/f"); string(b) != "a\nb\n" {
		t.Errorf("/f = %q", b)
	}
	if _, errb, rc := run(m, "x\n", "-o/nodir/f"); rc != exit.Fail || !strings.Contains(errb, "no such file") {
		t.Errorf("bad -o: rc %d %q", rc, errb)
	}
	var errb bytes.Buffer
	e := &fsx.Env{Args: []string{"sort"}, Stdin: strings.NewReader("x\n"), Stdout: failWriter{}, Stderr: &errb, FS: m, Cwd: "/"}
	if rc := Run(e); rc != exit.Fail || !strings.Contains(errb.String(), "write error") {
		t.Errorf("write error: rc %d %q", rc, errb.String())
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestSortUsage(t *testing.T) {
	m := fsx.NewMemFS()
	for _, c := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-z"}, "invalid option -- 'z'"},
		{[]string{"-k"}, "requires an argument -- 'k'"},
		{[]string{"-t"}, "requires an argument -- 't'"},
		{[]string{"-t", "ab"}, "one character"},
		{[]string{"-k", "0"}, "invalid key"},
		{[]string{"-k", "x"}, "invalid key"},
		{[]string{"-k", "1.0"}, "invalid key"},
		{[]string{"-k", "1.x"}, "invalid key"},
		{[]string{"-k", "1,y"}, "invalid key"},
	} {
		if _, errb, rc := run(m, "", c.args...); rc != exit.Usage || !strings.Contains(errb, c.msg) {
			t.Errorf("sort %q: rc %d %q, want %q", c.args, rc, errb, c.msg)
		}
	}
}

func TestCompareNum(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"-1", "1", -1}, {"1", "-1", 1}, {"-2", "-10", 1}, {"0", "-0", 0},
		{"12345678901234567890", "12345678901234567891", -1}, // beyond float64
		{"1.5", "1.25", 1}, {"  3", "3", 0}, {"x", "", 0},
	} {
		if got := compareNum(c.a, c.b); got != c.want {
			t.Errorf("compareNum(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
