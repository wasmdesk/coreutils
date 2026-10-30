// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sort

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// The inputs of the key tests.
var inputs = map[string]string{
	"a.txt": "b 2\na 10\nc 1\na 2\n",
	"p.txt": "x:1:30\ny:2:4\nz:3:100\nw:,:7\n",
	"m.txt": "x  b\ny a\nabcd\nxbad\nB\na-c\nab\nC\n -1\n10\n2.5\n-0\n0\nabc\n-2.50\n007\n",
}

// oracleCases are compared with the system sort (GNU on Linux, BSD on
// macOS) in the C locale: the reference, not expectations of ours.
var oracleCases = [][]string{
	{"-k2n", "a.txt"},
	{"-k2,2n", "-k1,1r", "a.txt"},
	{"-r", "-k2n", "a.txt"},
	{"-u", "-k1,1", "a.txt"},
	{"-t:", "-k3,3n", "p.txt"},
	{"-t", ":", "-k2,2", "p.txt"},
	{"-k2", "m.txt"},
	{"-k2b", "m.txt"},
	{"-k1.2", "m.txt"},
	{"-k1.2,1.3", "m.txt"},
	{"-f", "m.txt"},
	{"-d", "m.txt"},
	{"-n", "m.txt"},
	{"-nr", "m.txt"},
	{"-k1,1n", "m.txt"},
	{"-b", "-k2", "m.txt"},
	{"-k2,2", "-k1.2b,1.2", "m.txt"},
	{"-fu", "m.txt"},
	{"-i", "m.txt"},
	{"-k1.2n", "m.txt"},
	{"-k3", "m.txt"},
	{"a.txt", "p.txt", "m.txt"},
}

// benchInput is 200 000 lines "word number" in a scrambled order.
func benchInput() []byte {
	var b bytes.Buffer
	x := uint32(1)
	for range 200000 {
		x = x*1664525 + 1013904223
		fmt.Fprintf(&b, "w%05d %d\n", x>>20, x%1000003)
	}
	return b.Bytes()
}

// BenchmarkSortKeyN and BenchmarkSystemSortKeyN sort the same input with
// -k2n, ours and the system's (single process each; the system sort is
// given --parallel=1 where GNU supports it, through the C locale).
func BenchmarkSortKeyN(b *testing.B) {
	in := benchInput()
	m := fsx.NewMemFS()
	m.WriteFile("/in", in)
	for b.Loop() {
		Run(&fsx.Env{Args: []string{"sort", "-k2n", "/in"}, Stdout: io.Discard, Stderr: io.Discard, FS: m, Cwd: "/"})
	}
}

func BenchmarkSystemSortKeyN(b *testing.B) {
	ref, err := exec.LookPath("sort")
	if err != nil {
		b.Skip("no system sort")
	}
	p := filepath.Join(b.TempDir(), "in")
	os.WriteFile(p, benchInput(), 0o644)
	args := []string{"-k2n", p}
	if runtime.GOOS == "linux" {
		args = append([]string{"--parallel=1"}, args...)
	}
	for b.Loop() {
		cmd := exec.Command(ref, args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		if err := cmd.Run(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSortOracle(t *testing.T) {
	ref, err := exec.LookPath("sort")
	if err != nil || runtime.GOOS == "windows" {
		t.Skip("no POSIX sort to compare with")
	}
	dir := t.TempDir()
	m := fsx.NewMemFS()
	for name, data := range inputs {
		os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644)
		m.WriteFile("/"+name, []byte(data))
	}
	golden := map[string]string{}
	for _, args := range oracleCases {
		cmd := exec.Command(ref, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		want, err := cmd.Output()
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		golden[strings.Join(args, " ")] = string(want)
		var out, errb bytes.Buffer
		Run(&fsx.Env{Args: append([]string{"sort"}, args...), Stdout: &out, Stderr: &errb, FS: m, Cwd: "/"})
		if out.String() != string(want) {
			t.Errorf("sort %q:\n got %q\nwant %q (%s) %s", args, out.String(), want, ref, errb.String())
		}
	}
	// The committed golden file must be what this system sort says, so
	// that TestSortGolden (which runs everywhere) checks the reference.
	if os.Getenv("SORT_GOLDEN_WRITE") != "" {
		b, _ := json.MarshalIndent(golden, "", "  ")
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if have := readGolden(t); !maps.Equal(have, golden) {
		t.Errorf("%s differs from %s's output: regenerate with SORT_GOLDEN_WRITE=1", goldenPath, ref)
	}
}

const goldenPath = "testdata/oracle.golden"

func readGolden(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var g map[string]string
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// TestSortGolden checks the system sort's recorded answers on every
// platform, including those without a sort to ask (Windows).
func TestSortGolden(t *testing.T) {
	g := readGolden(t)
	m := fsx.NewMemFS()
	for name, data := range inputs {
		m.WriteFile("/"+name, []byte(data))
	}
	if len(g) != len(oracleCases) {
		t.Fatalf("%d golden answers for %d cases", len(g), len(oracleCases))
	}
	for _, args := range oracleCases {
		var out bytes.Buffer
		Run(&fsx.Env{Args: append([]string{"sort"}, args...), Stdout: &out, Stderr: &out, FS: m, Cwd: "/"})
		if want := g[strings.Join(args, " ")]; out.String() != want {
			t.Errorf("sort %q:\n got %q\nwant %q", args, out.String(), want)
		}
	}
}
