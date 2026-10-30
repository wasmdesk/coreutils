// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tail

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Wait is -f's pause between two looks at the files; it returns false to
// stop following (a seam for tests: the real one sleeps a second and
// never stops, as POSIX describes).
var Wait = func() bool {
	sleep(time.Second)
	return true
}

var sleep = time.Sleep

// Run is POSIX tail: tail [-f] [-c number | -n number] [file...]. A number
// counts from the end, or with + from the beginning (origin 1: +1 is the
// whole input); the default is -n 10. The obsolescent -number and +number
// forms mean -n. With several files each is preceded by "==> file <==".
// -f keeps following the FILEs, printing what is appended (and starting
// again from the beginning if a file shrinks); it is ignored when reading
// standard input.
func Run(env *fsx.Env) int {
	o, paths, err := parseArgs(env.Args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tail: %v\n", err)
		return exit.Usage
	}
	if len(paths) == 0 {
		paths = []string{fsx.StdinOperand}
	}
	rc := exit.Ok
	t := &tracker{env: env, multi: len(paths) > 1}
	type followed struct {
		name string
		off  int64
	}
	var follow []*followed
	for _, p := range paths {
		var data []byte
		var err error
		if p == fsx.StdinOperand {
			if env.Stdin != nil {
				data, err = io.ReadAll(env.Stdin)
			}
		} else {
			data, err = env.FS.ReadFile(fsx.Resolve(env.Cwd, p))
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "tail: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		t.header(p)
		if _, err := env.Stdout.Write(o.tail(data)); err != nil {
			fmt.Fprintf(env.Stderr, "tail: write error: %v\n", err)
			return exit.Fail
		}
		if o.follow && p != fsx.StdinOperand {
			follow = append(follow, &followed{p, int64(len(data))})
		}
	}
	for len(follow) > 0 && Wait() {
		for _, f := range follow {
			abs := fsx.Resolve(env.Cwd, f.name)
			st, err := env.FS.Stat(abs)
			if err != nil || st.Size == f.off {
				continue // gone for now, or nothing new
			}
			if st.Size < f.off {
				fmt.Fprintf(env.Stderr, "tail: %s: file truncated\n", f.name)
				f.off = 0
			}
			data, err := env.FS.ReadFile(abs)
			if err != nil || int64(len(data)) <= f.off {
				continue
			}
			t.header(f.name)
			if _, err := env.Stdout.Write(data[f.off:]); err != nil {
				return exit.Fail // the reader went away
			}
			f.off = int64(len(data))
		}
	}
	return rc
}

// tracker writes a "==> file <==" header when the file being shown
// changes (several files), with a blank line before all but the first.
type tracker struct {
	env   *fsx.Env
	multi bool
	last  string
	any   bool
}

func (t *tracker) header(name string) {
	if !t.multi || (t.any && t.last == name) {
		return
	}
	if t.any {
		fmt.Fprintln(t.env.Stdout)
	}
	fmt.Fprintf(t.env.Stdout, "==> %s <==\n", name)
	t.last, t.any = name, true
}

type options struct {
	n         int64
	fromStart bool
	bytes     bool
	follow    bool
}

func parseArgs(argv []string) (*options, []string, error) {
	o := &options{n: 10}
	if len(argv) > 0 {
		argv = argv[1:]
	}
	// The obsolescent first operand: -number or +number (lines).
	if len(argv) > 0 && len(argv[0]) > 1 && (argv[0][0] == '-' || argv[0][0] == '+') && isDigits(argv[0][1:]) {
		v, err := strconv.ParseInt(argv[0][1:], 10, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid number: %q", argv[0])
		}
		o.n, o.fromStart = v, argv[0][0] == '+'
		argv = argv[1:]
	}
	var paths []string
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "--" {
			paths = append(paths, argv[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			paths = append(paths, a)
			continue
		}
		for j := 1; j < len(a); j++ {
			switch c := a[j]; c {
			case 'f':
				o.follow = true
			case 'n', 'c':
				v := a[j+1:]
				if v == "" {
					i++
					if i >= len(argv) {
						return nil, nil, fmt.Errorf("option requires an argument -- '%c'", c)
					}
					v = argv[i]
				}
				if err := o.number(v); err != nil {
					return nil, nil, err
				}
				o.bytes = c == 'c'
				j = len(a)
			default:
				return nil, nil, fmt.Errorf("invalid option -- '%c'", c)
			}
		}
	}
	return o, paths, nil
}

// number reads [+-]digits; + counts from the beginning.
func (o *options) number(v string) error {
	s := v
	from := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		from, s = s[0] == '+', s[1:]
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || !isDigits(s) {
		return fmt.Errorf("invalid number: %q", v)
	}
	o.n, o.fromStart = n, from
	return nil
}

// isDigits reports whether s (never empty here) is all decimal digits.
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// tail returns the part of data to print.
func (o *options) tail(data []byte) []byte {
	if o.bytes {
		if o.fromStart {
			return data[min(max(o.n-1, 0), int64(len(data))):]
		}
		return data[int64(len(data))-min(o.n, int64(len(data))):]
	}
	if o.fromStart {
		// Skip n-1 lines.
		i := 0
		for k := int64(1); k < o.n && i < len(data); k++ {
			j := bytes.IndexByte(data[i:], '\n')
			if j < 0 {
				return nil
			}
			i += j + 1
		}
		return data[i:]
	}
	if o.n == 0 {
		return nil
	}
	// The last n lines: a final line without a newline counts.
	end := len(data)
	if end > 0 && data[end-1] == '\n' {
		end--
	}
	i := end
	for k := int64(0); k < o.n; k++ {
		j := bytes.LastIndexByte(data[:i], '\n')
		if j < 0 {
			return data
		}
		i = j
	}
	return data[i+1:]
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
