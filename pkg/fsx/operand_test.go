// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"strings"
	"testing"
)

func TestReadOperand(t *testing.T) {
	m := NewMemFS()
	_ = m.MkdirAll("/d")
	_ = m.WriteFile("/d/f", []byte("file"))
	e := &Env{Cwd: "/d", FS: m}
	if b, err := e.ReadOperand(StdinOperand); err != nil || b != nil {
		t.Errorf("nil stdin: %q %v", b, err)
	}
	e.Stdin = strings.NewReader("in")
	if b, err := e.ReadOperand("-"); err != nil || string(b) != "in" {
		t.Errorf("stdin: %q %v", b, err)
	}
	if b, err := e.ReadOperand("f"); err != nil || string(b) != "file" {
		t.Errorf("relative file: %q %v", b, err)
	}
	if _, err := e.ReadOperand("/nope"); err == nil {
		t.Error("missing file read")
	}
}
