// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunStatus(t *testing.T) {
	var errb bytes.Buffer
	e := &Env{Stderr: &errb}
	if code := e.RunStatus("xargs", Command{Name: "x"}); code != 126 || !strings.Contains(errb.String(), "cannot run programs here") {
		t.Errorf("no Exec: %d %q", code, errb.String())
	}
	for _, c := range []struct {
		code int
		err  error
		want int
		msg  string
	}{
		{3, nil, 3, ""},
		{0, ErrNoProgram, 127, "xargs: x: not found"},
		{0, errors.New("ENOEXEC"), 126, "xargs: x: ENOEXEC"},
	} {
		errb.Reset()
		e.Exec = func(Command) (int, error) { return c.code, c.err }
		if got := e.RunStatus("xargs", Command{Name: "x"}); got != c.want || !strings.Contains(errb.String(), c.msg) {
			t.Errorf("%v: %d %q", c.err, got, errb.String())
		}
	}
}
