// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package printf

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Cases outside the oracle (extensions, or where GNU and BSD differ).
func TestPrintfMore(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
		ok   bool
	}{
		{[]string{"--", "%s\n", "x"}, "x\n", true},
		{[]string{"%*d|%.*d|\n", "-4", "7", "-1", "8"}, "7   |8|\n", true}, // * negative: '-' flag; precision ignored
		{[]string{"%*s|\n", "99999999", "x"}, strings.Repeat(" ", MaxWidth-1) + "x|\n", true},
		{[]string{"%99999999d|\n", "1"}, strings.Repeat(" ", MaxWidth-1) + "1|\n", true},
		{[]string{"%.3s|%c|\n", "", ""}, "||\n", true},
		{[]string{"%d|%d\n", "'", "-9223372036854775808"}, "0|-9223372036854775808\n", true},
		{[]string{"%d\n", "-99999999999999999999"}, "-9223372036854775808\n", false},
		{[]string{"%d\n", "0x"}, "0\n", false},
		{[]string{"%f\n", "'"}, "0.000000\n", true},
		{[]string{"%f\n", "x"}, "0.000000\n", false},
		{[]string{"%b|\n", "\\0\\c"}, "\x00", true},         // GNU: \0 is a NUL byte
		{[]string{"%b|\n", "a\\q\\"}, "a\\q\\|\n", true},    // unknown and trailing backslash kept
		{[]string{"%b%s\n", "\\08", "|"}, "\x008|\n", true}, // \0ddd: 8 is not octal
		{[]string{"%3$s %1$s|\n", "a", "b"}, " a|\n", true}, // a missing numbered argument is empty
		{[]string{"x\\"}, "x\\", true},
		{[]string{"%s$|%d\n", "a", "0"}, "a$|0\n", true},                                    // a $ after a non-number is text
		{[]string{"%ld %lld %hhd %zd %jd\n", "1", "2", "3", "4", "5"}, "1 2 3 4 5\n", true}, // length modifiers: ignored (BSD refuses them)
		{[]string{"%e|%A\n", "-0", "inf"}, "-0.000000e+00|INF\n", true},
		// Where GNU and BSD differ: ours follows BSD (macOS) here.
		{[]string{"%c%c%c\n", "hello", "", "x"}, "hx\n", true},
		{[]string{"%q|\n"}, "", false},
		{[]string{"%2$s %1$s\n", "world", "hello"}, "hello world\n", true},
		{[]string{"%a|%A|%a|%.2a|%12a|%-12a|%012a|\n", "1.5", "-0.25", "1", "1", "2", "2", "3"},
			"0x1.8p+0|-0X1P-2|0x1p+0|0x1.00p+0|      0x1p+1|0x1p+1      |0x00001.8p+1|\n", true},
		{[]string{"%f\n", "1e500"}, "inf\n", false},
		{[]string{"%010.2f|% 08d|%-+5d|\n", "-3.14159", "42", "7"}, "-000003.14| 0000042|+7   |\n", true},
	} {
		var out, errb bytes.Buffer
		rc := Run(&fsx.Env{Args: append([]string{"printf"}, c.args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
		if out.String() != c.want || (rc == exit.Ok) != c.ok {
			t.Errorf("printf %q: %q rc %d %s, want %q ok %v", c.args, out.String(), rc, errb.String(), c.want, c.ok)
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestPrintfWriteError(t *testing.T) {
	var errb bytes.Buffer
	rc := Run(&fsx.Env{Args: []string{"printf", "x"}, Stdout: failWriter{}, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	if rc != exit.Fail || !strings.Contains(errb.String(), "write error") {
		t.Errorf("rc %d %q", rc, errb.String())
	}
}

func TestPad(t *testing.T) {
	for _, c := range []struct {
		s, flags string
		w        int
		want     string
	}{
		{"0x1p+0", "0", 9, "0x0001p+0"},
		{"-0x1p+0", "0", 9, "-0x001p+0"},
		{"12", "0", 4, "0012"},
		{"ab", "-", 4, "ab  "},
		{"ab", "", 4, "  ab"},
		{"abcd", "", 2, "abcd"},
	} {
		if got := pad(c.s, c.w, []byte(c.flags)); got != c.want {
			t.Errorf("pad(%q, %d, %q) = %q", c.s, c.w, c.flags, got)
		}
	}
}

// FuzzPrintf: any format and arguments end without a panic, and the
// output stays bounded.
func FuzzPrintf(f *testing.F) {
	for _, s := range []string{"%5.2f|%-*d|%b\\n", "%2$s%1$s", "%a%A%q", "\\0101\\c%%", "%*.*s"} {
		f.Add(s, "12", "-3", "a\\tb")
	}
	f.Fuzz(func(t *testing.T, format, a, b, c string) {
		var out, errb bytes.Buffer
		Run(&fsx.Env{Args: []string{"printf", format, a, b, c}, Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
		if limit := 16 * (MaxWidth + 64) * (len(format) + 1); out.Len() > limit {
			t.Fatalf("%d bytes of output", out.Len())
		}
	})
}
