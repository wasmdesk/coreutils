// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tr

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

const input = "Hello, World! 123 abc\tXYZ\n  spaced   out...\n\x01ctl\x7f\xe9\n"

// oracleCases are compared with the system tr in the C locale.
var oracleCases = [][]string{
	{"[:lower:]", "[:upper:]"},
	{"[:upper:]", "[:lower:]"},
	{"a-z", "A-Z"},
	{"-d", "[:digit:]"},
	{"-d", "[:punct:][:space:]"},
	{"-cd", "[:alnum:]\\n"},
	{"-s", "[:blank:]"},
	{"-s", " ", "_"},
	{"-c", "[:alnum:]", "[_*]"},
	{"-ds", "[:digit:]", "[:space:]"},
	{"abc", "[x*]"},
	{"a-f", "[1*3]Z"},
	{"[:xdigit:]", "[.*]"},
	{"\\101\\n", "a_"},
	{"lo", "x"},
	{"[:cntrl:]", "[?*]"},
	{"[:graph:]", "[g*]"},
	{"[:print:]", "[p*]"},
	{"-d", "[:alpha:]"},
	{"-s", "lo"},
	{"\\t\\\\", "TB"},
	{"\\a\\b\\f\\r\\v\\001", "abfrvC"},
	// Not here: [=c=] (BSD expands it differently in the C locale; POSIX
	// locale: c itself) and -C (BSD keeps bytes outside its collation).
}

type answer struct {
	Out string
	OK  bool
}

// entry is an answer as stored: the output as bytes (base64 in JSON),
// since output need not be valid UTF-8 and JSON strings must be.
type entry struct {
	Out []byte `json:"out"`
	OK  bool   `json:"ok"`
}

const goldenPath = "testdata/oracle.golden"

func ours(args []string) answer {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"tr"}, args...), Stdin: strings.NewReader(input), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return answer{out.String(), rc == exit.Ok}
}

func TestTrOracle(t *testing.T) {
	ref, err := exec.LookPath("tr")
	if err != nil || runtime.GOOS == "windows" {
		t.Skip("no system tr to compare with")
	}
	golden := map[string]answer{}
	for _, args := range oracleCases {
		cmd := exec.Command(ref, args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		want := answer{string(out), err == nil}
		golden[strings.Join(args, "\x00")] = want
		if got := ours(args); got != want {
			t.Errorf("tr %q:\n got %+v\nwant %+v", args, got, want)
		}
	}
	if os.Getenv("TR_GOLDEN_WRITE") != "" {
		stored := map[string]entry{}
		for k, a := range golden {
			stored[k] = entry{[]byte(a.Out), a.OK}
		}
		b, _ := json.MarshalIndent(stored, "", "  ")
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if have := readGolden(t); !maps.Equal(have, golden) {
		t.Errorf("%s differs from the system tr: regenerate with TR_GOLDEN_WRITE=1", goldenPath)
	}
}

func readGolden(t *testing.T) map[string]answer {
	t.Helper()
	b, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]entry
	if err := json.Unmarshal(b, &stored); err != nil {
		t.Fatal(err)
	}
	g := map[string]answer{}
	for k, e := range stored {
		g[k] = answer{string(e.Out), e.OK}
	}
	return g
}

func TestTrGolden(t *testing.T) {
	g := readGolden(t)
	if len(g) != len(oracleCases) {
		t.Fatalf("%d golden answers for %d cases", len(g), len(oracleCases))
	}
	for _, args := range oracleCases {
		if got, want := ours(args), g[strings.Join(args, "\x00")]; got != want {
			t.Errorf("tr %q:\n got %+v\nwant %+v", args, got, want)
		}
	}
}

func TestTrErrors(t *testing.T) {
	for _, c := range []struct {
		args []string
		msg  string
	}{
		{[]string{"-x", "a"}, "invalid option -- 'x'"},
		{[]string{"[:nope:]", "a"}, `invalid character class "nope"`},
		{[]string{"a", "[:digit:]"}, "only character classes that may appear in string2"},
		{[]string{"a-z", "[:upper:]"}, "misaligned [:upper:]"},
		{[]string{"-c", "[:lower:]", "[:upper:]"}, "misaligned"},
		{[]string{"[a*3]", "b"}, "only allowed in string2"},
		{[]string{"a", "[b*x]"}, "invalid repeat count"},
		{[]string{"a", "[b*9999999]"}, "too large"},
		{[]string{"\\777", "a"}, "not a byte"},
		{[]string{"a", ""}, "string2 must be non-empty"},
		{[]string{"-d", "a", "b"}, "expected 1 operand"},
		{[]string{"b-a", "x"}, "reverse order"},
		{[]string{"-ds", "a", "[:nope:]"}, "invalid character class"},
		{[]string{"a", "b-a"}, "reverse order"},
		{[]string{"a-\\777", "b"}, "not a byte"},
		{[]string{"a", "\\777"}, "not a byte"},
	} {
		var out, errb bytes.Buffer
		rc := Run(&fsx.Env{Args: append([]string{"tr"}, c.args...), Stdin: strings.NewReader("x"), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
		if rc != exit.Usage || !strings.Contains(errb.String(), c.msg) {
			t.Errorf("tr %q: rc %d %q, want %q", c.args, rc, errb.String(), c.msg)
		}
	}
}

// More forms, outside the oracle.
func TestTrMore(t *testing.T) {
	for _, c := range []struct {
		args     []string
		in, want string
	}{
		{[]string{"--", "-a", "+b"}, "a-c", "b+c"},
		{[]string{"a", "[b*010]"}, "a", "b"}, // octal count
		{[]string{"ab", "[x*0]"}, "abc", "xxc"},
		{[]string{"a\\", "xy"}, "a\\", "xy"},   // a lone trailing backslash
		{[]string{"\\q", "x"}, "q", "x"},       // an unknown escape: the character
		{[]string{"[ab", "xyz"}, "[ab", "xyz"}, // an unterminated bracket is text
		{[]string{"[:x", "ab"}, "[:x", "abb"},
		{[]string{"[=a", "bcd"}, "[=a", "bcd"},
		{[]string{"[a*", "xyz"}, "[a*", "xyz"},
		{[]string{"[=l=]o", "Lq"}, "lol", "LqL"},            // POSIX locale: [=l=] is l
		{[]string{"-C", "-d", "a-z"}, "ab1\xe9c", "abc"},    // -C is -c in the POSIX locale
		{[]string{"-c", "a-y", "[z*]"}, "ab\xe9\n", "abzz"}, // complement in byte order
	} {
		var out, errb bytes.Buffer
		rc := Run(&fsx.Env{Args: append([]string{"tr"}, c.args...), Stdin: strings.NewReader(c.in), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
		if out.String() != c.want || rc != exit.Ok {
			t.Errorf("tr %q <<< %q: %q rc %d %s, want %q", c.args, c.in, out.String(), rc, errb.String(), c.want)
		}
	}
}

// chunks returns its parts one Read at a time: -s holds across reads.
type chunks struct{ parts []string }

func (c *chunks) Read(p []byte) (int, error) {
	if len(c.parts) == 0 {
		return 0, errors.New("EIO")
	}
	n := copy(p, c.parts[0])
	c.parts = c.parts[1:]
	return n, nil
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("EPIPE") }

func TestTrStreaming(t *testing.T) {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: []string{"tr", "-s", "a"}, Stdin: &chunks{[]string{"baa", "aab", ""}}, Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	if out.String() != "bab" || rc != exit.Fail || !strings.Contains(errb.String(), "EIO") {
		t.Errorf("out %q rc %d err %q", out.String(), rc, errb.String())
	}
	errb.Reset()
	rc = Run(&fsx.Env{Args: []string{"tr", "a", "b"}, Stdin: strings.NewReader("a"), Stdout: failWriter{}, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	if rc != exit.Fail || !strings.Contains(errb.String(), "write error") {
		t.Errorf("write error: rc %d %q", rc, errb.String())
	}
}

var benchIn = bytes.Repeat([]byte(input), 1<<18) // 15 MB

// BenchmarkTrUpper and BenchmarkSystemTrUpper: [:lower:] -> [:upper:] on
// the same 15 MB, ours and the system's (one process each).
func BenchmarkTrUpper(b *testing.B) {
	for b.Loop() {
		Run(&fsx.Env{Args: []string{"tr", "[:lower:]", "[:upper:]"}, Stdin: bytes.NewReader(benchIn), Stdout: io.Discard, Stderr: io.Discard, FS: fsx.NewMemFS(), Cwd: "/"})
	}
}

func BenchmarkSystemTrUpper(b *testing.B) {
	ref, err := exec.LookPath("tr")
	if err != nil {
		b.Skip("no system tr")
	}
	for b.Loop() {
		cmd := exec.Command(ref, "[:lower:]", "[:upper:]")
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = bytes.NewReader(benchIn)
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

// FuzzTr: any operands and input end without a panic.
func FuzzTr(f *testing.F) {
	f.Add("[:lower:]", "[:upper:]", "Hello")
	f.Add("a-z\\n", "[x*]", "abc\n")
	f.Add("[=a=][b*3]", "\\101-\\132", "zz")
	f.Fuzz(func(t *testing.T, s1, s2, in string) {
		var out bytes.Buffer
		for _, args := range [][]string{{s1, s2}, {"-d", s1}, {"-s", s1}, {"-cds", s1, s2}} {
			out.Reset()
			Run(&fsx.Env{Args: append([]string{"tr"}, args...), Stdin: strings.NewReader(in), Stdout: &out, Stderr: &out, FS: fsx.NewMemFS(), Cwd: "/"})
		}
	})
}
