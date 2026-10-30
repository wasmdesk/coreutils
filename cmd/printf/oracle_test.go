// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package printf

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// oracleCases are argument lists compared with the system printf: what
// POSIX defines, so that GNU and BSD agree.
var oracleCases = [][]string{
	{"%5d|%-5d|%05d|%+d|% d\n", "42", "42", "42", "42", "42"},
	{"%.3d|%8.3d|%x|%X|%#x|%o|%#o\n", "7", "7", "255", "255", "255", "8", "8"},
	{"%d %d %d %d\n", "0x1f", "010", "-12", "+5"},
	{"%d %d\n", "'A", "\"z"},
	{"%u %x\n", "-1", "-1"},
	{"%s|%10s|%-10s|%.2s|\n", "abc", "abc", "abc", "abc"},
	{"%c%c\n", "hello", "x"},
	{"%f|%.2f|%10.3f|%-10.1f|%e|%E\n", "3.14159", "3.14159", "3.14159", "3.14159", "1234.5", "0.000123"},
	{"%g|%g|%g|%G|%.3g|%#g\n", "100000", "1000000", "0.0001", "1e-10", "3.14159", "2"},
	{"%s-%s\n", "a", "b", "c", "d", "e"},
	{"%s %d|\n"},
	{"%b|%b|%b\n", "a\\tb", "\\0101", "x\\\\y"}, // not \0: BSD stops at the NUL, GNU writes it
	{"%b", "stop\\chere", "never"},
	{"a\\tb\\101\\0\\n"},
	{"%%|\n"},
	{"%*d|%-*d|%.*f\n", "5", "42", "4", "7", "2", "3.14159"},
	{"%d\n", "12abc"},
	{"%d\n", "abc"},
	{"%i|%5.1s|\n", "-0", "xyz"},
	{"done%"},
	{"%f|%F|%e|%G|%5.1f|%-6f|%+f|% f|\n", "inf", "-inf", "nan", "INF", "inf", "inf", "inf", "inf"},
	{"%d\n", "99999999999999999999"},
	{"%d %d %x\n", "0XFF", "-0x10", "0Xa"},
	{"%f %.0f %d\n", "'A", "2.5", "'"},
	{"\\a\\b\\f\\r\\v|\\\"|\\101\\1012|\n"},
	// Not here, as GNU and BSD differ (ours: flags_test.go): %c of "" (GNU
	// writes a NUL), %q (a GNU extension), %n$ (not in GNU's utility),
	// %a's normalisation (glibc: 0xcp-3), and 1e500 (GNU: long double).
}

func systemPrintf(t *testing.T, args []string) (string, bool, bool) {
	t.Helper()
	ref, err := exec.LookPath("printf")
	if err != nil || runtime.GOOS == "windows" {
		return "", false, false
	}
	cmd := exec.Command(ref, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	return out.String(), err == nil, true
}

type answer struct {
	Out string `json:"out"`
	OK  bool   `json:"ok"`
}

const goldenPath = "testdata/oracle.golden"

func ours(args []string) answer {
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"printf"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return answer{out.String(), rc == 0}
}

func TestPrintfOracle(t *testing.T) {
	if _, _, ok := systemPrintf(t, []string{""}); !ok {
		t.Skip("no system printf to compare with")
	}
	golden := map[string]answer{}
	for _, args := range oracleCases {
		out, ok, _ := systemPrintf(t, args)
		want := answer{out, ok}
		golden[strings.Join(args, "\x00")] = want
		if got := ours(args); got != want {
			t.Errorf("printf %q:\n got %+v\nwant %+v", args, got, want)
		}
	}
	if os.Getenv("PRINTF_GOLDEN_WRITE") != "" {
		b, _ := json.MarshalIndent(golden, "", "  ")
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if have := readGolden(t); !maps.Equal(have, golden) {
		t.Errorf("%s differs from the system printf: regenerate with PRINTF_GOLDEN_WRITE=1", goldenPath)
	}
}

func readGolden(t *testing.T) map[string]answer {
	t.Helper()
	b, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var g map[string]answer
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// TestPrintfGolden checks the recorded answers on every platform.
func TestPrintfGolden(t *testing.T) {
	g := readGolden(t)
	if len(g) != len(oracleCases) {
		t.Fatalf("%d golden answers for %d cases", len(g), len(oracleCases))
	}
	for _, args := range oracleCases {
		if got, want := ours(args), g[strings.Join(args, "\x00")]; got != want {
			t.Errorf("printf %q:\n got %+v\nwant %+v", args, got, want)
		}
	}
}
