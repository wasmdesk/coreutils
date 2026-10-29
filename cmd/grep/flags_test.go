// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grep

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// tree is a small directory tree for the recursive cases.
func tree() *fsx.MemFS {
	m := fsx.NewMemFS()
	m.MkdirAll("/src/sub")
	m.WriteFile("/src/a.txt", []byte("alpha\nfoo bar\nfoobar\n"))
	m.WriteFile("/src/sub/b.txt", []byte("foo\nbaz\nfoo foo\n"))
	m.WriteFile("/src/c.txt", []byte("nothing\n"))
	return m
}

func grep(m *fsx.MemFS, cwd, stdin string, args ...string) (string, string, int) {
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: append([]string{"grep"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: cwd}
	if stdin != "" {
		e.Stdin = strings.NewReader(stdin)
	}
	rc := Run(e)
	return out.String(), errb.String(), rc
}

func TestGrepFlags(t *testing.T) {
	cases := []struct {
		name, cwd, stdin string
		args             []string
		out              string
		rc               int
	}{
		{"recursive", "/", "", []string{"-r", "foo", "/src"}, "/src/a.txt:foo bar\n/src/a.txt:foobar\n/src/sub/b.txt:foo\n/src/sub/b.txt:foo foo\n", exit.Ok},
		{"recursive from cwd, -R, -n", "/src", "", []string{"-Rn", "baz"}, "./sub/b.txt:2:baz\n", exit.Ok},
		{"recursive -h", "/", "", []string{"-rh", "baz", "/src/"}, "baz\n", exit.Ok},
		{"files only", "/", "", []string{"-rl", "foo", "/src"}, "/src/a.txt\n/src/sub/b.txt\n", exit.Ok},
		{"count", "/", "", []string{"-c", "foo", "/src/a.txt"}, "2\n", exit.Ok},
		{"count, several files", "/", "", []string{"-c", "foo", "/src/a.txt", "/src/c.txt"}, "/src/a.txt:2\n/src/c.txt:0\n", exit.Fail - exit.Fail},
		{"count -v", "/", "", []string{"-cv", "foo", "/src/a.txt"}, "1\n", exit.Ok},
		{"only matching", "/", "", []string{"-o", "foo", "/src/sub/b.txt"}, "foo\nfoo\nfoo\n", exit.Ok},
		{"only matching -E", "/", "", []string{"-oE", "o+", "/src/a.txt"}, "oo\noo\n", exit.Ok},
		{"only matching -v prints nothing", "/", "", []string{"-ov", "foo", "/src/a.txt"}, "", exit.Ok},
		{"only matching empty -E match", "/", "a\n", []string{"-oE", "x*"}, "", exit.Ok},
		{"quiet match", "/", "", []string{"-q", "foo", "/src/a.txt"}, "", exit.Ok},
		{"quiet no match", "/", "", []string{"-q", "zzz", "/src/a.txt"}, "", exit.Fail},
		{"quiet hides errors, finds later", "/", "", []string{"-q", "foo", "/nope", "/src/a.txt"}, "", exit.Ok},
		{"quiet -l", "/", "", []string{"-ql", "foo", "/src/a.txt"}, "", exit.Ok},
		{"word literal", "/", "", []string{"-w", "foo", "/src/a.txt"}, "foo bar\n", exit.Ok},
		{"word literal, second occurrence", "/", "xfoo foo\n", []string{"-w", "foo"}, "xfoo foo\n", exit.Ok},
		{"word literal, none", "/", "foox\n", []string{"-w", "foo"}, "", exit.Fail},
		{"word -E", "/", "", []string{"-wE", "fo+", "/src/a.txt"}, "foo bar\n", exit.Ok},
		{"word -E -i -o", "/", "FOO x\n", []string{"-woiE", "fo+"}, "FOO\n", exit.Ok},
		{"fixed after -E", "/", "a.c\nabc\n", []string{"-EF", "a.c"}, "a.c\n", exit.Ok},
		{"-H on one file", "/", "", []string{"-H", "baz", "/src/sub/b.txt"}, "/src/sub/b.txt:baz\n", exit.Ok},
		{"stdin name with -H", "/", "x\n", []string{"-H", "x"}, "(standard input):x\n", exit.Ok},
		{"--", "/", "-v\n", []string{"--", "-v"}, "-v\n", exit.Ok},
		{"empty pattern", "/", "a\n\nb\n", []string{""}, "a\n\nb\n", exit.Ok},
		{"ignore case, word", "/", "Foo\n", []string{"-iw", "foo"}, "Foo\n", exit.Ok},
		{"missing file only", "/", "", []string{"x", "/nope"}, "", exit.Usage},
		{"bad option", "/", "", []string{"-Z", "x"}, "", exit.Usage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, errs, rc := grep(tree(), c.cwd, c.stdin, c.args...)
			if out != c.out || rc != c.rc {
				t.Errorf("grep %q = %q %d (stderr %q), want %q %d", c.args, out, rc, errs, c.out, c.rc)
			}
		})
	}
}

func TestGrepRecursiveDepth(t *testing.T) {
	m := fsx.NewMemFS()
	p := ""
	for i := 0; i < MaxDepth+3; i++ {
		p += "/d"
	}
	m.MkdirAll(p)
	m.WriteFile(p+"/deep", []byte("x\n"))
	m.WriteFile("/d/top", []byte("x\n"))
	out, _, _ := grep(m, "/", "", "-r", "x", "/d")
	if out != "/d/top:x\n" {
		t.Errorf("%q", out)
	}
	// A recursive walk that reaches stdin or an unreadable directory.
	if out, _, rc := grep(m, "/", "x\n", "-r", "x", "-"); out != "(standard input):x\n" || rc != exit.Ok {
		t.Errorf("stdin under -r: %q %d", out, rc)
	}
}

// unlistable is a file system whose directories cannot be listed.
type unlistable struct{ *fsx.MemFS }

func (unlistable) ReadDir(string) ([]fsx.FileInfo, error) { return nil, fsx.ErrNotDir }

func TestGrepRecursiveUnlistable(t *testing.T) {
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: []string{"grep", "-r", "foo", "/src"}, Stdout: &out, Stderr: &errb, FS: unlistable{tree()}, Cwd: "/"}
	// The directory is kept as an operand, so reading it reports the error.
	if rc := Run(e); rc != exit.Usage || out.Len() != 0 || !strings.Contains(errb.String(), "/src") {
		t.Errorf("rc=%d out=%q err=%q", rc, out.String(), errb.String())
	}
}
