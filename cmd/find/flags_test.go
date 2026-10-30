// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package find

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

var t0 = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

// tree: /w/{a.txt (100 B, 0 days), big.log (2000 B, 3 days), d/{b.TXT (0 B,
// 10 days), e/{c (513 B)}}}, cwd /w.
func tree(t *testing.T) *fsx.MemFS {
	t.Helper()
	defer func(n func() time.Time) { fsx.Now = n }(fsx.Now)
	fsx.Now = func() time.Time { return t0 }
	m := fsx.NewMemFS()
	m.MkdirAll("/w/d/e")
	m.WriteFile("/w/a.txt", make([]byte, 100))
	m.WriteFile("/w/big.log", make([]byte, 2000))
	m.WriteFile("/w/d/b.TXT", nil)
	m.WriteFile("/w/d/e/c", make([]byte, 513))
	m.Chtimes("/w/big.log", t0.Add(-3*24*time.Hour-time.Hour))
	m.Chtimes("/w/d/b.TXT", t0.Add(-10*24*time.Hour))
	return m
}

func find(t *testing.T, m *fsx.MemFS, args ...string) (string, string, int) {
	t.Helper()
	defer func(n func() time.Time) { Now = n }(Now)
	Now = func() time.Time { return t0 }
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"find"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/w"})
	return out.String(), errb.String(), rc
}

func TestFindPrimaries(t *testing.T) {
	m := tree(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		// Paths are formed from the operand as given (POSIX).
		{nil, ".\n./a.txt\n./big.log\n./d\n./d/b.TXT\n./d/e\n./d/e/c\n"},
		{[]string{"d/"}, "d/\nd/b.TXT\nd/e\nd/e/c\n"},
		{[]string{".", "-type", "d"}, ".\n./d\n./d/e\n"},
		{[]string{".", "-type", "f", "-name", "*.txt"}, "./a.txt\n"},
		{[]string{".", "-iname", "*.txt"}, "./a.txt\n./d/b.TXT\n"},
		{[]string{".", "-type", "l"}, ""},
		{[]string{".", "-maxdepth", "1"}, ".\n./a.txt\n./big.log\n./d\n"},
		{[]string{".", "-maxdepth", "0"}, ".\n"},
		{[]string{".", "-mindepth", "2", "-type", "f"}, "./d/b.TXT\n./d/e/c\n"},
		// -size: 512-byte blocks rounded up; c bytes; k; +/-.
		{[]string{".", "-type", "f", "-size", "1"}, "./a.txt\n"},
		{[]string{".", "-type", "f", "-size", "2"}, "./d/e/c\n"},
		{[]string{".", "-type", "f", "-size", "+2"}, "./big.log\n"},
		{[]string{".", "-type", "f", "-size", "-1"}, "./d/b.TXT\n"},
		{[]string{".", "-size", "100c"}, "./a.txt\n"},
		{[]string{".", "-type", "f", "-size", "-1k"}, "./d/b.TXT\n"},
		{[]string{".", "-type", "f", "-size", "+1k"}, "./big.log\n"},
		// -mtime: whole days, remainder discarded.
		{[]string{".", "-type", "f", "-mtime", "0"}, "./a.txt\n./d/e/c\n"},
		{[]string{".", "-type", "f", "-mtime", "3"}, "./big.log\n"},
		{[]string{".", "-type", "f", "-mtime", "+3"}, "./d/b.TXT\n"},
		{[]string{".", "-type", "f", "-mtime", "-4"}, "./a.txt\n./big.log\n./d/e/c\n"},
		{[]string{".", "-type", "f", "-newer", "big.log"}, "./a.txt\n./d/e/c\n"},
		// Operators, precedence, -prune, -print.
		{[]string{".", "-name", "*.txt", "-o", "-name", "c"}, "./a.txt\n./d/e/c\n"},
		{[]string{".", "-type", "f", "!", "-name", "*.*"}, "./d/e/c\n"},
		{[]string{".", "-type", "f", "-not", "(", "-name", "*.txt", "-or", "-name", "*.log", ")"}, "./d/b.TXT\n./d/e/c\n"},
		{[]string{".", "-name", "d", "-prune", "-o", "-type", "f", "-print"}, "./a.txt\n./big.log\n"},
		{[]string{".", "-name", "c", "-a", "-print", "-print"}, "./d/e/c\n./d/e/c\n"},
		{[]string{".", "-false", "-o", "-true", "-maxdepth", "0"}, ".\n"},
	} {
		out, errb, rc := find(t, m, c.args...)
		if out != c.want || rc != exit.Ok {
			t.Errorf("find %q:\n got %q rc %d %s\nwant %q", c.args, out, rc, errb, c.want)
		}
	}
}

func TestFindUsage(t *testing.T) {
	m := tree(t)
	for _, c := range []struct {
		args []string
		msg  string
	}{
		{[]string{".", "-bogus"}, "unknown predicate `-bogus'"},
		{[]string{".", "-exec", "rm", ";"}, "unknown predicate `-exec'"},
		{[]string{".", "-type", "x"}, "unknown type"},
		{[]string{".", "-type"}, "missing argument"},
		{[]string{".", "-size", "x"}, "-size: invalid number"},
		{[]string{".", "-size"}, "missing argument"},
		{[]string{".", "-size", "+-1"}, "invalid number"},
		{[]string{".", "-mtime", "1.5"}, "-mtime: invalid number"},
		{[]string{".", "-mtime"}, "missing argument"},
		{[]string{".", "-newer", "nope"}, "nope: no such file"},
		{[]string{".", "-newer"}, "missing argument"},
		{[]string{".", "-maxdepth", "-1"}, "not a non-negative integer"},
		{[]string{".", "-mindepth"}, "missing argument"},
		{[]string{".", "-iname"}, "missing argument"},
		{[]string{".", "(", "-true"}, "missing `)'"},
		{[]string{".", "(", ")"}, "unknown predicate `)'"},
		{[]string{".", "-true", ")"}, "unexpected `)'"},
		{[]string{".", "-true", "-o"}, "expression expected"},
		{[]string{".", "!"}, "expression expected"},
	} {
		out, errb, rc := find(t, m, c.args...)
		if rc != exit.Usage || out != "" || !strings.Contains(errb, c.msg) {
			t.Errorf("find %q: rc %d out %q err %q, want %q", c.args, rc, out, errb, c.msg)
		}
	}
}

// A symbolic-link loop (OSFS follows links) is cut at MaxDepth.
type loopFS struct{ fsx.FS }

func (loopFS) Stat(string) (fsx.FileInfo, error) { return fsx.FileInfo{Name: "l", IsDir: true}, nil }
func (loopFS) ReadDir(string) ([]fsx.FileInfo, error) {
	return []fsx.FileInfo{{Name: "l", IsDir: true}}, nil
}

func TestFindDepthBound(t *testing.T) {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: []string{"find", "/l", "-name", "nothing"}, Stdout: &out, Stderr: &errb, FS: loopFS{fsx.NewMemFS()}, Cwd: "/"})
	if rc != exit.Fail || !strings.Contains(errb.String(), "too many levels") {
		t.Errorf("rc %d err %q", rc, errb.String())
	}
}

// The default clock is the real one: a file written now is 0 days old.
func TestFindRealClock(t *testing.T) {
	m := fsx.NewMemFS()
	m.WriteFile("/f", nil)
	var out bytes.Buffer
	Run(&fsx.Env{Args: []string{"find", "/f", "-mtime", "0"}, Stdout: &out, Stderr: &out, FS: m, Cwd: "/"})
	if out.String() != "/f\n" {
		t.Errorf("%q", out.String())
	}
}
