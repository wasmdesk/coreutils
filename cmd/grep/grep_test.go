// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package grep

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func env(t *testing.T, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer, *fsx.MemFS) {
	t.Helper()
	m := fsx.NewMemFS()
	var out, errb bytes.Buffer
	return &fsx.Env{Args: append([]string{"grep"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"}, &out, &errb, m
}

func TestGrepMatch(t *testing.T) {
	e, out, _, m := env(t, "hi", "/f")
	_ = m.WriteFile("/f", []byte("hi\nbye\nhi there\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "hi\nhi there\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestGrepNoMatch(t *testing.T) {
	e, out, _, m := env(t, "zzz", "/f")
	_ = m.WriteFile("/f", []byte("hi\nbye\n"))
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if out.Len() != 0 {
		t.Errorf("out = %q", out.String())
	}
}

func TestGrepIgnoreCase(t *testing.T) {
	e, out, _, m := env(t, "-i", "HI", "/f")
	_ = m.WriteFile("/f", []byte("hi\nbye\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "hi\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestGrepInvert(t *testing.T) {
	e, out, _, m := env(t, "-v", "hi", "/f")
	_ = m.WriteFile("/f", []byte("hi\nbye\nhi there\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "bye\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestGrepNumbered(t *testing.T) {
	e, out, _, m := env(t, "-n", "hi", "/f")
	_ = m.WriteFile("/f", []byte("a\nhi\nb\nhi\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "2:hi\n4:hi\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestGrepMultiFile(t *testing.T) {
	e, out, _, m := env(t, "hi", "/a", "/b")
	_ = m.WriteFile("/a", []byte("hi\n"))
	_ = m.WriteFile("/b", []byte("hi there\n"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	got := out.String()
	if !strings.Contains(got, "/a:hi\n") || !strings.Contains(got, "/b:hi there\n") {
		t.Errorf("out = %q", got)
	}
}

func TestGrepNoTrailingNewline(t *testing.T) {
	e, out, _, m := env(t, "hi", "/f")
	_ = m.WriteFile("/f", []byte("hi"))
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got, want := out.String(), "hi\n"; got != want {
		t.Errorf("out = %q", got)
	}
}

func TestGrepNoArgs(t *testing.T) {
	e, _, errb, _ := env(t)
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestGrepEmptyArgv(t *testing.T) {
	e := &fsx.Env{Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
}

func TestGrepPatternOnly(t *testing.T) {
	e, _, errb, _ := env(t, "pat")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestGrepMissingFile(t *testing.T) {
	e, _, errb, _ := env(t, "x", "/nope")
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "no such file or directory") {
		t.Errorf("stderr = %q", errb.String())
	}
}

// -E activates the go-ruby-regexp engine. The cases below cover the regex
// feature surface a user would reach for at a shell (alternation, char
// class, anchors, quantifiers) plus the three flag combinations the rest of
// grep already supports.
func TestGrepRegexCases(t *testing.T) {
	cases := []struct {
		name string
		args []string
		body string
		want string
		rc   int
	}{
		{
			name: "alternation",
			args: []string{"-E", "a|b", "/f"},
			body: "alpha\nbeta\ngamma\n",
			want: "alpha\nbeta\ngamma\n", // every line contains 'a'
			rc:   exit.Ok,
		},
		{
			name: "char-class",
			args: []string{"-E", "^[abc]", "/f"},
			body: "alpha\nbeta\ngamma\ndelta\n",
			want: "alpha\nbeta\n",
			rc:   exit.Ok,
		},
		{
			name: "anchor-start",
			args: []string{"-E", "^foo", "/f"},
			body: "foo\nfoobar\nbarfoo\n",
			want: "foo\nfoobar\n",
			rc:   exit.Ok,
		},
		{
			name: "anchor-end",
			args: []string{"-E", "bar$", "/f"},
			body: "foo\nfoobar\nbarfoo\n",
			want: "foobar\n",
			rc:   exit.Ok,
		},
		{
			name: "quantifier-plus",
			args: []string{"-E", "a+", "/f"},
			body: "alpha\nbeta\ngamma\nzzz\n",
			want: "alpha\nbeta\ngamma\n",
			rc:   exit.Ok,
		},
		{
			name: "quantifier-star",
			args: []string{"-E", "^ab*$", "/f"},
			body: "a\nab\nabb\nx\n",
			want: "a\nab\nabb\n",
			rc:   exit.Ok,
		},
		{
			name: "ignore-case",
			args: []string{"-E", "-i", "ALPHA|BETA", "/f"},
			body: "alpha\nbeta\ngamma\n",
			want: "alpha\nbeta\n",
			rc:   exit.Ok,
		},
		{
			name: "invert",
			args: []string{"-E", "-v", "^[ab]", "/f"},
			body: "alpha\nbeta\ngamma\ndelta\n",
			want: "gamma\ndelta\n",
			rc:   exit.Ok,
		},
		{
			name: "numbered",
			args: []string{"-E", "-n", "a+", "/f"},
			body: "x\nalpha\ny\ngamma\n",
			want: "2:alpha\n4:gamma\n",
			rc:   exit.Ok,
		},
		{
			name: "no-match",
			args: []string{"-E", "^zzz$", "/f"},
			body: "alpha\nbeta\n",
			want: "",
			rc:   exit.Fail,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, out, _, m := env(t, tc.args...)
			_ = m.WriteFile("/f", []byte(tc.body))
			if rc := Run(e); rc != tc.rc {
				t.Fatalf("rc = %d, want %d", rc, tc.rc)
			}
			if got := out.String(); got != tc.want {
				t.Errorf("out = %q, want %q", got, tc.want)
			}
		})
	}
}

// An invalid -E pattern must surface to stderr and exit 2 (usage), not
// silently match nothing.
func TestGrepRegexInvalid(t *testing.T) {
	e, _, errb, m := env(t, "-E", "[", "/f")
	_ = m.WriteFile("/f", []byte("alpha\n"))
	if rc := Run(e); rc != exit.Usage {
		t.Fatalf("rc = %d, want %d", rc, exit.Usage)
	}
	if !strings.Contains(errb.String(), "invalid regex") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestGrepPrettyErr(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{fsx.ErrNotFound, "no such file or directory"},
		{fsx.ErrIsDir, "is a directory"},
		{errors.New("boom"), "boom"},
	}
	for _, c := range cases {
		if got := prettyErr(c.err); got != c.want {
			t.Errorf("prettyErr(%v) = %q", c.err, got)
		}
	}
}
