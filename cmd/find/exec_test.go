// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package find

import (
	"bytes"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// execFind runs find over a small tree with a fake Exec that records each
// command line and exits 1 for paths containing "bad".
func execFind(t *testing.T, args ...string) ([]string, string, string, int) {
	t.Helper()
	m := fsx.NewMemFS()
	m.MkdirAll("/w/d")
	m.WriteFile("/w/a", nil)
	m.WriteFile("/w/bad", nil)
	m.WriteFile("/w/d/c", nil)
	var calls []string
	var out, errb bytes.Buffer
	e := &fsx.Env{Args: append([]string{"find"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/w",
		Exec: func(c fsx.Command) (int, error) {
			calls = append(calls, strings.Join(c.Args, " "))
			switch {
			case c.Name == "missing":
				return 0, fsx.ErrNoProgram
			case strings.Contains(strings.Join(c.Args, " "), "bad"):
				return 1, nil
			}
			return 0, nil
		}}
	rc := Run(e)
	return calls, out.String(), errb.String(), rc
}

func TestFindExec(t *testing.T) {
	// ; : once per path, {} replaced (also inside an argument), no implicit -print.
	calls, out, _, rc := execFind(t, ".", "-type", "f", "-exec", "rm", "-f", "{}", "x{}y", ";")
	if strings.Join(calls, "|") != "rm -f ./a x./ay|rm -f ./bad x./bady|rm -f ./d/c x./d/cy" || out != "" || rc != exit.Ok {
		t.Errorf("; : %q %q %d", calls, out, rc)
	}
	// -exec is a test: its status selects for -print.
	calls, out, _, rc = execFind(t, ".", "-type", "f", "-exec", "check", "{}", ";", "-print")
	if len(calls) != 3 || out != "./a\n./d/c\n" || rc != exit.Ok {
		t.Errorf("as a test: %q %q %d", calls, out, rc)
	}
	// + : one command with all the paths; a failure makes the status 1.
	calls, _, _, rc = execFind(t, ".", "-type", "f", "-exec", "ls", "-l", "{}", "+")
	if strings.Join(calls, "|") != "ls -l ./a ./bad ./d/c" || rc != exit.Fail {
		t.Errorf("+ : %q %d", calls, rc)
	}
	calls, _, _, rc = execFind(t, ".", "-name", "a", "-exec", "ls", "{}", "+")
	if strings.Join(calls, "|") != "ls ./a" || rc != exit.Ok {
		t.Errorf("+ ok: %q %d", calls, rc)
	}
	// + with nothing selected runs nothing.
	if calls, _, _, _ := execFind(t, ".", "-name", "zzz", "-exec", "ls", "{}", "+"); len(calls) != 0 {
		t.Errorf("empty +: %q", calls)
	}
	// A program that is not found: reported, false, status 1.
	_, out, errb, rc := execFind(t, ".", "-name", "a", "-exec", "missing", "{}", ";")
	if out != "" || !strings.Contains(errb, "find: missing: not found") || rc != exit.Fail {
		t.Errorf("missing: %q %q %d", out, errb, rc)
	}
	// {} that is not last before + is an argument, and + an ordinary one then.
	calls, _, _, _ = execFind(t, ".", "-name", "a", "-exec", "echo", "{}", "x", "+", ";")
	if strings.Join(calls, "|") != "echo ./a x +" {
		t.Errorf("not a + form: %q", calls)
	}
}

// Batches are cut at MaxExecSize.
func TestFindExecBatchSize(t *testing.T) {
	m := fsx.NewMemFS()
	m.Mkdir("/w")
	long := strings.Repeat("n", 1000)
	for i := range 300 {
		m.WriteFile("/w/"+long+string(rune('a'+i%26))+strings.Repeat("z", i/26), nil)
	}
	n := 0
	e := &fsx.Env{Args: []string{"find", "/w", "-type", "f", "-exec", "x", "{}", "+"}, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, FS: m, Cwd: "/",
		Exec: func(c fsx.Command) (int, error) {
			n++
			size := 0
			for _, a := range c.Args[1:] {
				size += len(a) + 1
			}
			if size > MaxExecSize {
				t.Errorf("batch of %d bytes", size)
			}
			return 0, nil
		}}
	if rc := Run(e); rc != exit.Ok || n < 3 {
		t.Errorf("rc %d, %d batches", rc, n)
	}
}
