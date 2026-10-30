// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package strftime

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// instants exercise the week numbering edges: a Sunday that is ISO week 53
// of the previous year, new year's day, a leap day, the epoch.
var instants = []int64{
	0,          // 1970-01-01 Thursday
	1793351107, // 2026-10-30 09:05:07 Friday
	1609632000, // 2021-01-03 00:00:00 Sunday, ISO 2020-W53
	1798761599, // 2026-12-31 23:59:59 Thursday
	951782400,  // 2000-02-29 00:00:00 Tuesday
	946684799,  // 1999-12-31 23:59:59 Friday
	1704067200, // 2024-01-01 00:00:00 Monday
	1230768000, // 2009-01-01 00:00:00 Thursday, ISO 2009-W01
	1262217600, // 2009-12-31 Thursday, ISO 2009-W53
}

// posixLayouts are POSIX conversions the system date must agree on.
var posixLayouts = []string{
	"%a %A %b %B %h",
	"%c",
	"%C %y %Y %G %g",
	"%d %e %j %m",
	"%D %F %x",
	"%H %I %M %S %p %r %R %T %X",
	"%u %w %U %W %V",
	"%n%t%%",
	"%s",
	"%z %Z",
	"plain text %% 100%",
}

// systemDate formats seconds with the system date in UTC, C locale.
func systemDate(t *testing.T, secs int64, layout string) (string, bool) {
	t.Helper()
	ref, err := exec.LookPath("date")
	if err != nil || runtime.GOOS == "windows" {
		return "", false
	}
	for _, args := range [][]string{
		{"-u", "-d", fmt.Sprintf("@%d", secs), "+" + layout}, // GNU
		{"-u", "-r", fmt.Sprint(secs), "+" + layout},         // BSD
	} {
		cmd := exec.Command(ref, args...)
		cmd.Env = append(os.Environ(), "LC_ALL=C", "TZ=UTC")
		if out, err := cmd.Output(); err == nil {
			return strings.TrimSuffix(string(out), "\n"), true
		}
	}
	return "", false
}

const goldenPath = "testdata/posix.golden"

func key(secs int64, layout string) string { return fmt.Sprintf("%d %s", secs, layout) }

// TestOracle compares with the system date and checks that the committed
// golden answers are the system's (SORT-style: TestGolden runs everywhere).
func TestOracle(t *testing.T) {
	if _, ok := systemDate(t, 0, "%Y"); !ok {
		t.Skip("no system date to compare with")
	}
	golden := map[string]string{}
	for _, secs := range instants {
		for _, l := range posixLayouts {
			want, _ := systemDate(t, secs, l)
			golden[key(secs, l)] = want
			if got := Format(time.Unix(secs, 0).UTC(), l); got != want {
				t.Errorf("%d %q: got %q, system %q", secs, l, got, want)
			}
		}
	}
	if os.Getenv("STRFTIME_GOLDEN_WRITE") != "" {
		b, _ := json.MarshalIndent(golden, "", "  ")
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if have := readGolden(t); !maps.Equal(have, golden) {
		t.Errorf("%s differs from the system date: regenerate with STRFTIME_GOLDEN_WRITE=1", goldenPath)
	}
}

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

func TestGolden(t *testing.T) {
	g := readGolden(t)
	if len(g) != len(instants)*len(posixLayouts) {
		t.Fatalf("%d golden answers", len(g))
	}
	for _, secs := range instants {
		for _, l := range posixLayouts {
			if got, want := Format(time.Unix(secs, 0).UTC(), l), g[key(secs, l)]; got != want {
				t.Errorf("%d %q: got %q, want %q", secs, l, got, want)
			}
		}
	}
}

// Flags, widths and the GNU extensions (man7 date(1), POSIX strftime's
// '+' flag rules).
func TestFlags(t *testing.T) {
	tm := time.Date(2026, 3, 5, 7, 8, 9, 12345, time.UTC)
	for _, c := range []struct{ layout, want string }{
		{"%-d/%-m %-H", "5/3 7"},
		{"%_d|%_m|%_H", " 5| 3| 7"},
		{"%0e", "05"},
		{"%k|%l|%P", " 7| 7|am"},
		{"%N", "000012345"},
		{"%^a %^B", "THU MARCH"},
		{"%#a %#p %#Z", "THU am utc"},
		{"%10A|%010d|%5d|%05a", "  Thursday|0000000005|00005|00Thu"},
		{"%:z", "+00:00"},
		{"%+Y|%+6Y|%+C|%+4C", "2026|+02026|20|+020"},
		// %+12F: the year as %+6Y (POSIX: width x-6), five digits: signed.
		{"%+F|%+12F|%+11F|%12F|%3F", "2026-03-05|+02026-03-05|+2026-03-05|002026-03-05|2026-03-05"},
		{"%EY %Od %Ey", "2026 05 26"},
		{"%q %", "%q %"},
		{"%-", "%-"},
		{"%5", "%5"},
		{"%99999999999d", strings.Repeat("0", 1023) + "5"},
		{"%-y %_y", "26 26"},
	} {
		if got := Format(tm, c.layout); got != c.want {
			t.Errorf("%q: got %q, want %q", c.layout, got, c.want)
		}
	}
}

func TestZonesAndSigns(t *testing.T) {
	east := time.FixedZone("CET", 3600)
	west := time.FixedZone("XYZ", -(5*3600 + 30*60))
	if got := Format(time.Date(2026, 1, 1, 13, 0, 0, 0, east), "%z %:z %Z %I %p"); got != "+0100 +01:00 CET 01 PM" {
		t.Errorf("east: %q", got)
	}
	if got := Format(time.Date(2026, 1, 1, 0, 0, 0, 0, west), "%z %I %r"); got != "-0530 12 12:00:00 AM" {
		t.Errorf("west: %q", got)
	}
	bc := time.Date(-45, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := Format(bc, "%Y|%+Y|%5Y|%_5Y|%-Y"); got != "-45|-45|-0045|  -45|-45" {
		t.Errorf("negative year: %q", got)
	}
	if got := Format(time.Unix(-1, 0).UTC(), "%s"); got != "-1" {
		t.Errorf("before the epoch: %q", got)
	}
}

// FuzzFormat: any layout formats without a panic, and text without a %
// is copied unchanged.
func FuzzFormat(f *testing.F) {
	for _, s := range []string{"%Y-%m-%d", "%+12F", "%_5Y%:z%", "%E%O%-_0^#+9q", "plain"} {
		f.Add(s, int64(1793351107))
	}
	f.Fuzz(func(t *testing.T, layout string, secs int64) {
		out := Format(time.Unix(secs%(1<<40), 0).UTC(), layout)
		if !strings.Contains(layout, "%") && out != layout {
			t.Fatalf("%q -> %q", layout, out)
		}
		if len(out) > 1100*len(layout)+64 {
			t.Fatalf("output %d bytes for a %d-byte layout", len(out), len(layout))
		}
	})
}
