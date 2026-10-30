// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package head

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(stdin io.Reader, args ...string) (string, string, int) {
	m := fsx.NewMemFS()
	m.WriteFile("/a", []byte("1\n2\n3\n"))
	m.WriteFile("/b", []byte("xyz"))
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"head"}, args...), Stdin: stdin, Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"})
	return out.String(), errb.String(), rc
}

func TestHeadFlags(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"-c", "2", "/a"}, "1\n"},
		{[]string{"-c2", "/b"}, "xy"},
		{[]string{"-c", "0", "/a"}, ""},
		{[]string{"-c", "99", "/b"}, "xyz"},
		{[]string{"-n1", "/a"}, "1\n"},
		{[]string{"-2", "/a"}, "1\n2\n"},
		{[]string{"-n", "5", "/b"}, "xyz"},
		{[]string{"-c1", "/a", "/b"}, "==> /a <==\n1\n==> /b <==\nx"},
		{[]string{"-n1", "--", "/a"}, "1\n"},
	} {
		out, errb, rc := run(nil, c.args...)
		if out != c.want || rc != exit.Ok {
			t.Errorf("head %q: %q rc %d %s, want %q", c.args, out, rc, errb, c.want)
		}
	}
	if _, errb, rc := run(nil, "-x"); rc != exit.Usage || !strings.Contains(errb, "invalid option -- 'x'") {
		t.Errorf("-x: %d %q", rc, errb)
	}
	if _, errb, rc := run(nil, "-c"); rc != exit.Usage || !strings.Contains(errb, "requires an argument -- 'c'") {
		t.Errorf("-c: %d %q", rc, errb)
	}
}

// endless is standard input that never ends: head must stop reading.
type endless struct{ read int }

func (e *endless) Read(p []byte) (int, error) {
	e.read += len(p)
	for i := range p {
		p[i] = 'y'
		if i%2 == 1 {
			p[i] = '\n'
		}
	}
	return len(p), nil
}

func TestHeadStopsReading(t *testing.T) {
	in := &endless{}
	if out, _, rc := run(in, "-n", "3"); out != "yy\ny\ny\n" && out != "y\ny\ny\n" || rc != exit.Ok {
		t.Errorf("lines: %q %d", out, rc)
	}
	if in.read > 1<<16 {
		t.Errorf("read %d bytes for 3 lines", in.read)
	}
	if out, _, _ := run(&endless{}, "-c", "3"); out != "y\ny" {
		t.Errorf("bytes: %q", out)
	}
}

// A line longer than bufio's buffer is copied whole as one line.
func TestHeadLongLine(t *testing.T) {
	long := strings.Repeat("z", 10000) + "\nnext\n"
	if out, _, _ := run(strings.NewReader(long), "-n", "1"); out != long[:10001] {
		t.Errorf("len %d", len(out))
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("EIO") }

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestHeadIOErrors(t *testing.T) {
	if _, errb, rc := run(failReader{}); rc != exit.Fail || !strings.Contains(errb, "EIO") {
		t.Errorf("read: %d %q", rc, errb)
	}
	var errb bytes.Buffer
	rc := Run(&fsx.Env{Args: []string{"head"}, Stdin: strings.NewReader("a\n"), Stdout: failWriter{}, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	if rc != exit.Fail || !strings.Contains(errb.String(), "EPIPE") {
		t.Errorf("write: %d %q", rc, errb.String())
	}
	if out, _, rc := run(nil); out != "" || rc != exit.Ok {
		t.Errorf("no stdin: %q %d", out, rc)
	}
}
