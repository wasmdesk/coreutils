// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tail

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(m *fsx.MemFS, stdin string, args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: append([]string{"tail"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}
	if stdin != "" {
		e.Stdin = strings.NewReader(stdin)
	}
	rc := Run(e)
	return out.String(), errb.String(), rc
}

func TestTailFlags(t *testing.T) {
	m := fsx.NewMemFS()
	m.WriteFile("/a", []byte("1\n2\n3\n4\n"))
	m.WriteFile("/b", []byte("x\ny"))
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"-n", "2", "/a"}, "3\n4\n"},
		{[]string{"-n", "-1", "/a"}, "4\n"},
		{[]string{"-n", "+3", "/a"}, "3\n4\n"},
		{[]string{"-n", "+1", "/a"}, "1\n2\n3\n4\n"},
		{[]string{"-n", "+0", "/a"}, "1\n2\n3\n4\n"}, // not conforming; read as +1
		{[]string{"-n", "+9", "/a"}, ""},
		{[]string{"-n", "+3", "/b"}, ""}, // past the end, no final newline
		{[]string{"-n", "0", "/a"}, ""},
		{[]string{"-n9", "/a"}, "1\n2\n3\n4\n"},
		{[]string{"-n1", "/b"}, "y"}, // a last line without a newline
		{[]string{"-n2", "/b"}, "x\ny"},
		{[]string{"-c", "3", "/a"}, "\n4\n"},
		{[]string{"-c", "+7", "/a"}, "4\n"},
		{[]string{"-c", "+0", "/a"}, "1\n2\n3\n4\n"},
		{[]string{"-c99", "/b"}, "x\ny"},
		{[]string{"-c", "+99", "/b"}, ""},
		{[]string{"-2", "/a"}, "3\n4\n"},
		{[]string{"+4", "/a"}, "4\n"},
		{[]string{"-c1", "/a", "/b"}, "==> /a <==\n\n\n==> /b <==\ny"},
		{[]string{"--", "/b"}, "x\ny"},
		{[]string{"-f", "-n1"}, "s\n"}, // -f on standard input: ignored
	} {
		out, errb, rc := run(m, "r\ns\n", c.args...)
		if out != c.want || rc != exit.Ok {
			t.Errorf("tail %q: %q rc %d %s, want %q", c.args, out, rc, errb, c.want)
		}
	}
	for _, c := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-x"}, "invalid option -- 'x'"},
		{[]string{"-c"}, "requires an argument -- 'c'"},
		{[]string{"-99999999999999999999"}, "invalid number"},
	} {
		if _, errb, rc := run(m, "", c.args...); rc != exit.Usage || !strings.Contains(errb, c.msg) {
			t.Errorf("tail %q: rc %d %q, want %q", c.args, rc, errb, c.msg)
		}
	}
}

// -f prints what is appended, says when a file shrinks, and stops when
// Wait says so.
func TestTailFollow(t *testing.T) {
	defer func(w func() bool) { Wait = w }(Wait)
	m := fsx.NewMemFS()
	m.WriteFile("/log", []byte("a\nb\n"))
	m.WriteFile("/other", []byte("o\n"))
	steps := []func(){
		func() { m.WriteFile("/log", []byte("a\nb\nc\n")) }, // appended
		func() {}, // nothing new
		func() { m.WriteFile("/log", []byte("Z\n")) },      // truncated
		func() { m.Remove("/log") },                        // gone for now
		func() { m.WriteFile("/other", []byte("o\np\n")) }, // the other file
		func() { m.WriteFile("/log", []byte("Z\nnew\n")) }, // back, grown
	}
	i := 0
	Wait = func() bool {
		if i == len(steps) {
			return false
		}
		steps[i]()
		i++
		return true
	}
	out, errb, rc := run(m, "", "-f", "-n1", "/log", "/other")
	want := "==> /log <==\nb\n\n==> /other <==\no\n\n==> /log <==\nc\nZ\n\n==> /other <==\np\n\n==> /log <==\nnew\n"
	if out != want || rc != exit.Ok || !strings.Contains(errb, "/log: file truncated") {
		t.Errorf("out\n%q\nwant\n%q\nerr %q rc %d", out, want, errb, rc)
	}
}

// fewerFS: Stat says the file grew, ReadFile disagrees (a race).
type fewerFS struct{ *fsx.MemFS }

func (f fewerFS) Stat(p string) (fsx.FileInfo, error) {
	st, err := f.MemFS.Stat(p)
	st.Size += 10
	return st, err
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestTailFollowEdges(t *testing.T) {
	defer func(w func() bool) { Wait = w }(Wait)
	m := fsx.NewMemFS()
	m.WriteFile("/f", []byte("x\n"))
	n := 0
	Wait = func() bool { n++; return n < 3 }
	var out bytes.Buffer
	if rc := Run(&fsx.Env{Args: []string{"tail", "-f", "/f"}, Stdout: &out, Stderr: &out, FS: fewerFS{m}, Cwd: "/"}); rc != exit.Ok || out.String() != "x\n" {
		t.Errorf("race: %d %q", rc, out.String())
	}
	// The reader goes away while following.
	n = 0
	Wait = func() bool { m.WriteFile("/f", []byte("x\ny\n")); n++; return n < 3 }
	w := &failAfter{ok: 1}
	if rc := Run(&fsx.Env{Args: []string{"tail", "-f", "/f"}, Stdout: w, Stderr: &out, FS: m, Cwd: "/"}); rc != exit.Fail {
		t.Errorf("EPIPE while following: %d", rc)
	}
	// A write error on the first output.
	var errb bytes.Buffer
	if rc := Run(&fsx.Env{Args: []string{"tail", "/f"}, Stdout: failWriter{}, Stderr: &errb, FS: m, Cwd: "/"}); rc != exit.Fail || !strings.Contains(errb.String(), "write error") {
		t.Errorf("write error: %d %q", rc, errb.String())
	}
}

// The real Wait pauses one second and never stops (POSIX's loop).
func TestTailWaitInterval(t *testing.T) {
	defer func(s func(time.Duration)) { sleep = s }(sleep)
	var slept time.Duration
	sleep = func(d time.Duration) { slept += d }
	if !Wait() || slept != time.Second {
		t.Errorf("slept %v", slept)
	}
}

// failAfter accepts ok writes, then fails.
type failAfter struct{ ok int }

func (f *failAfter) Write(p []byte) (int, error) {
	if f.ok == 0 {
		return 0, errors.New("EPIPE")
	}
	f.ok--
	return len(p), nil
}
