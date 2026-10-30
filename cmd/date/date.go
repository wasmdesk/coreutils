// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package date

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
	"github.com/wasmdesk/coreutils/pkg/strftime"
)

// Now is the clock seam. Defaults to time.Now; tests swap it in to render
// deterministic output without sneaking around the public Run signature.
var Now = time.Now

// Local is the zone used without -u (TZ, read by Go at start); a seam.
var Local = func() *time.Location { return time.Local }

// DefaultFormat is POSIX date's output with no operand (POSIX locale).
const DefaultFormat = "%a %b %e %H:%M:%S %Z %Y"

// Run is POSIX date: date [-u] [+format], writing the time with the
// strftime conversions of pkg/strftime (POSIX, plus GNU's %N %P %k %l %:z
// and - _ ^ # flags). -u uses UTC; otherwise the local zone (TZ). -d
// STRING (GNU) shows another time: @SECONDS, RFC 3339, or
// "YYYY-MM-DD[ HH:MM[:SS]]" in the zone in use.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	utc, layout := false, DefaultFormat
	var dateArg string
	haveDate := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-u":
			utc = true
		case a == "-d" || strings.HasPrefix(a, "-d"):
			v := strings.TrimPrefix(a, "-d")
			if v == "" {
				i++
				if i >= len(args) {
					fmt.Fprintln(env.Stderr, "date: option requires an argument -- 'd'")
					return exit.Usage
				}
				v = args[i]
			}
			dateArg, haveDate = v, true
		case strings.HasPrefix(a, "+"):
			layout = a[1:]
		default:
			fmt.Fprintf(env.Stderr, "date: invalid argument: %q (setting the clock is not supported)\n", a)
			return exit.Usage
		}
	}
	loc := Local()
	if utc {
		loc = time.UTC
	}
	when := Now().In(loc)
	if haveDate {
		t, err := parse(dateArg, loc)
		if err != nil {
			fmt.Fprintf(env.Stderr, "date: invalid date: %q\n", dateArg)
			return exit.Fail
		}
		when = t.In(loc)
	}
	fmt.Fprintln(env.Stdout, strftime.Format(when, layout))
	return exit.Ok
}

// parse reads -d's STRING.
func parse(s string, loc *time.Location) (time.Time, error) {
	if rest, ok := strings.CutPrefix(s, "@"); ok {
		secs, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(secs, 0), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	var err error
	for _, l := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		var t time.Time
		if t, err = time.ParseInLocation(l, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}
