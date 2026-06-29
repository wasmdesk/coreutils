// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tr

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func newEnv(stdin string, args ...string) (*fsx.Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	return &fsx.Env{
		Args:   append([]string{"tr"}, args...),
		Stdin:  strings.NewReader(stdin),
		Stdout: &out, Stderr: &errb,
		FS:  fsx.NewMemFS(),
		Cwd: "/",
	}, &out, &errb
}

func TestTrTranslate(t *testing.T) {
	e, out, _ := newEnv("hello", "abcdefghijklmnopqrstuvwxyz", "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if rc := Run(e); rc != exit.Ok {
		t.Fatalf("rc = %d", rc)
	}
	if got := out.String(); got != "HELLO" {
		t.Errorf("got %q", got)
	}
}

func TestTrTranslateRange(t *testing.T) {
	e, out, _ := newEnv("hello", "a-z", "A-Z")
	_ = Run(e)
	if got := out.String(); got != "HELLO" {
		t.Errorf("got %q", got)
	}
}

func TestTrTranslateShortSet2(t *testing.T) {
	// SET2 shorter -> last byte repeats. "ab" -> "x"; "axx" expected from "abb".
	e, out, _ := newEnv("abb", "ab", "x")
	_ = Run(e)
	if got := out.String(); got != "xxx" {
		t.Errorf("got %q", got)
	}
}

func TestTrDelete(t *testing.T) {
	e, out, _ := newEnv("hello world", "-d", "lo")
	_ = Run(e)
	if got := out.String(); got != "he wrd" {
		t.Errorf("got %q", got)
	}
}

func TestTrSqueeze(t *testing.T) {
	e, out, _ := newEnv("aaabbbcccd", "-s", "abc")
	_ = Run(e)
	if got := out.String(); got != "abcd" {
		t.Errorf("got %q", got)
	}
}

func TestTrDeleteSqueeze(t *testing.T) {
	e, out, _ := newEnv("aaabbbcccd", "-d", "-s", "ac")
	_ = Run(e)
	// After delete of a,c: "bbbd". Squeeze of {a,c} doesn't affect bbbd.
	if got := out.String(); got != "bbbd" {
		t.Errorf("got %q", got)
	}
}

func TestTrTranslateSqueeze(t *testing.T) {
	// xlat a->b first, then squeeze b's. "aab" -> "bbb" -> "b".
	e, out, _ := newEnv("aab", "-s", "a", "b")
	_ = Run(e)
	if got := out.String(); got != "b" {
		t.Errorf("got %q", got)
	}
}

func TestTrEscape(t *testing.T) {
	e, out, _ := newEnv("a\nb", "\\n", "X")
	_ = Run(e)
	if got := out.String(); got != "aXb" {
		t.Errorf("got %q", got)
	}
	e2, out2, _ := newEnv("a\tb", "\\t", "X")
	_ = Run(e2)
	if got := out2.String(); got != "aXb" {
		t.Errorf("got %q", got)
	}
	e3, out3, _ := newEnv("a\\b", "\\\\", "X")
	_ = Run(e3)
	if got := out3.String(); got != "aXb" {
		t.Errorf("got %q", got)
	}
}

// Reversed range "z-a" is treated as a literal "z", "-", "a" sequence.
func TestTrReversedRange(t *testing.T) {
	e, out, _ := newEnv("z-a", "z-a", "XYZ")
	_ = Run(e)
	if got := out.String(); got != "XYZ" {
		t.Errorf("got %q", got)
	}
}

func TestTrDeleteBadArity(t *testing.T) {
	e, _, errb := newEnv("", "-d")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "-d takes exactly one") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestTrSqueezeBadArity(t *testing.T) {
	e, _, errb := newEnv("", "-s")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "one or two SET") {
		t.Errorf("stderr = %q", errb.String())
	}
}

// `tr -s` with 3+ operands is rejected.
func TestTrSqueezeTooManyOperands(t *testing.T) {
	e, _, errb := newEnv("", "-s", "a", "b", "c")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "one or two SET") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestTrTranslateBadArity(t *testing.T) {
	e, _, errb := newEnv("", "ab")
	if rc := Run(e); rc != exit.Usage {
		t.Errorf("rc = %d", rc)
	}
	if !strings.Contains(errb.String(), "two SETs") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestTrStdinNil(t *testing.T) {
	e := &fsx.Env{Args: []string{"tr", "a", "b"}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Ok {
		t.Errorf("rc = %d", rc)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestTrStdinReadError(t *testing.T) {
	e := &fsx.Env{Args: []string{"tr", "a", "b"}, Stdin: errReader{}, Stdout: new(bytes.Buffer), Stderr: new(bytes.Buffer), FS: fsx.NewMemFS(), Cwd: "/"}
	if rc := Run(e); rc != exit.Fail {
		t.Errorf("rc = %d", rc)
	}
}

func TestTrSqueezeEmpty(t *testing.T) {
	e, out, _ := newEnv("", "-s", "a")
	_ = Run(e)
	if out.String() != "" {
		t.Errorf("got %q", out.String())
	}
}

// Trailing escape backslash with nothing after is preserved literally.
func TestTrTrailingBackslash(t *testing.T) {
	e, out, _ := newEnv(`\`, `\`, "X")
	_ = Run(e)
	if got := out.String(); got != "X" {
		t.Errorf("got %q", got)
	}
}
