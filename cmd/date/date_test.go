// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package date

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	defer func(n func() time.Time, l func() *time.Location) { Now, Local = n, l }(Now, Local)
	Now = func() time.Time { return time.Date(2026, 6, 29, 12, 34, 56, 789, time.UTC) }
	cest := time.FixedZone("CEST", 2*3600)
	Local = func() *time.Location { return cest }
	var out, errb bytes.Buffer
	rc := Run(&fsx.Env{Args: append([]string{"date"}, args...), Stdout: &out, Stderr: &errb, FS: fsx.NewMemFS(), Cwd: "/"})
	return out.String(), errb.String(), rc
}

func TestDate(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "Mon Jun 29 14:34:56 CEST 2026\n"}, // POSIX default, local zone
		{[]string{"-u"}, "Mon Jun 29 12:34:56 UTC 2026\n"},
		{[]string{"+%Y-%m-%d %H:%M"}, "2026-06-29 14:34\n"},
		{[]string{"-u", "+%s.%N"}, "1782736496.000000789\n"},
		{[]string{"-u", "-d", "@0", "+%F %T"}, "1970-01-01 00:00:00\n"},
		{[]string{"-d@86400", "-u", "+%A"}, "Friday\n"},
		{[]string{"-u", "-d", "2020-01-02T03:04:05Z", "+%c"}, "Thu Jan  2 03:04:05 2020\n"},
		{[]string{"-d", "2020-01-02 03:04", "+%T %Z %z"}, "03:04:00 CEST +0200\n"},
		{[]string{"-d", "2020-01-02", "-u", "+%F %T"}, "2020-01-02 00:00:00\n"}, // -u: as if TZ=UTC0, for parsing too
		{[]string{"-d", "2020-01-02 03:04:05", "+%-d/%-m %I%P"}, "2/1 03am\n"},
		{[]string{"+"}, "\n"},
	} {
		out, errb, rc := run(t, c.args...)
		if out != c.want || rc != exit.Ok {
			t.Errorf("date %q: %q rc %d %s, want %q", c.args, out, rc, errb, c.want)
		}
	}
}

func TestDateErrors(t *testing.T) {
	for _, c := range []struct {
		args []string
		rc   int
		msg  string
	}{
		{[]string{"-d"}, exit.Usage, "requires an argument -- 'd'"},
		{[]string{"-d", "yesterday"}, exit.Fail, `invalid date: "yesterday"`},
		{[]string{"-d", "@x"}, exit.Fail, "invalid date"},
		{[]string{"--bogus"}, exit.Usage, "invalid argument"},
		{[]string{"0101000026"}, exit.Usage, "setting the clock is not supported"},
	} {
		out, errb, rc := run(t, c.args...)
		if rc != c.rc || out != "" || !strings.Contains(errb, c.msg) {
			t.Errorf("date %q: rc %d out %q err %q", c.args, rc, out, errb)
		}
	}
}

// The real seams: the local zone and the clock.
func TestDateDefaults(t *testing.T) {
	if Local() != time.Local {
		t.Error("Local is not time.Local")
	}
	var out bytes.Buffer
	Run(&fsx.Env{Args: []string{"date", "-u", "+%Y"}, Stdout: &out, Stderr: &out, FS: fsx.NewMemFS(), Cwd: "/"})
	if y := strings.TrimSpace(out.String()); len(y) != 4 || y < "2026" {
		t.Errorf("year %q", y)
	}
}
