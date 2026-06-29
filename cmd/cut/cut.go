// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package cut

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Mode chooses what each LIST entry counts (fields, characters, bytes). We
// treat -c and -b identically (v0 is ASCII; multi-byte support lands with the
// utf8 SIMD work).
type Mode int

const (
	modeFields Mode = iota
	modeChars
)

// Run extracts portions of each line. Exactly one of -f / -c / -b must be
// given; -f also accepts -d DELIM (default TAB).
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var mode Mode = -1
	var list string
	delim := "\t"
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-f", strings.HasPrefix(a, "-f"):
			val, err := optValue(a, args, &i, "-f")
			if err != "" {
				fmt.Fprintln(env.Stderr, err)
				return exit.Usage
			}
			mode = modeFields
			list = val
		case a == "-c", strings.HasPrefix(a, "-c"):
			val, err := optValue(a, args, &i, "-c")
			if err != "" {
				fmt.Fprintln(env.Stderr, err)
				return exit.Usage
			}
			mode = modeChars
			list = val
		case a == "-b", strings.HasPrefix(a, "-b"):
			val, err := optValue(a, args, &i, "-b")
			if err != "" {
				fmt.Fprintln(env.Stderr, err)
				return exit.Usage
			}
			mode = modeChars
			list = val
		case a == "-d", strings.HasPrefix(a, "-d"):
			val, err := optValue(a, args, &i, "-d")
			if err != "" {
				fmt.Fprintln(env.Stderr, err)
				return exit.Usage
			}
			delim = val
		default:
			paths = append(paths, a)
		}
	}
	if mode == -1 {
		fmt.Fprintln(env.Stderr, "cut: one of -f / -c / -b is required")
		return exit.Usage
	}
	ranges, err := parseList(list)
	if err != nil {
		fmt.Fprintf(env.Stderr, "cut: invalid list: %s\n", err)
		return exit.Usage
	}
	data, rc := slurp(env, paths)
	if rc != exit.Ok {
		return rc
	}
	lines := splitLines(data)
	for _, l := range lines {
		fmt.Fprintln(env.Stdout, cutLine(l, ranges, mode, delim))
	}
	return exit.Ok
}

// rng is one entry from LIST: [lo,hi] inclusive, 1-based. hi==0 means
// "open-ended" (extends to the end of the line).
type rng struct {
	lo, hi int
}

// optValue returns the value for a flag that can be either glued (`-c1`) or
// separated (`-c 1`). Returns ("", "<message>") on a missing operand.
func optValue(a string, args []string, i *int, name string) (string, string) {
	if len(a) > len(name) {
		return a[len(name):], ""
	}
	*i++
	if *i >= len(args) {
		return "", "cut: option requires an argument: " + name
	}
	return args[*i], ""
}

func parseList(list string) ([]rng, error) {
	if list == "" {
		return nil, errors.New("empty list")
	}
	var out []rng
	for _, part := range strings.Split(list, ",") {
		if part == "" {
			return nil, errors.New("empty range")
		}
		dash := strings.Index(part, "-")
		if dash < 0 {
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("bad number: %q", part)
			}
			out = append(out, rng{lo: n, hi: n})
			continue
		}
		left := part[:dash]
		right := part[dash+1:]
		lo, hi := 1, 0
		if left != "" {
			v, err := strconv.Atoi(left)
			if err != nil || v < 1 {
				return nil, fmt.Errorf("bad number: %q", left)
			}
			lo = v
		}
		if right != "" {
			v, err := strconv.Atoi(right)
			if err != nil || v < 1 {
				return nil, fmt.Errorf("bad number: %q", right)
			}
			hi = v
		}
		out = append(out, rng{lo: lo, hi: hi})
	}
	return out, nil
}

// cutLine projects a single input line through the supplied ranges.
func cutLine(line string, ranges []rng, mode Mode, delim string) string {
	if mode == modeChars {
		var b strings.Builder
		for _, r := range ranges {
			lo := r.lo
			hi := r.hi
			if hi == 0 || hi > len(line) {
				hi = len(line)
			}
			if lo > len(line) || lo > hi {
				continue
			}
			b.WriteString(line[lo-1 : hi])
		}
		return b.String()
	}
	// modeFields
	fields := strings.Split(line, delim)
	if !strings.Contains(line, delim) {
		// No delimiter present -> emit the unmodified line (matches GNU cut
		// without --only-delimited).
		return line
	}
	var out []string
	for _, r := range ranges {
		lo := r.lo
		hi := r.hi
		if hi == 0 || hi > len(fields) {
			hi = len(fields)
		}
		if lo > len(fields) || lo > hi {
			continue
		}
		out = append(out, fields[lo-1:hi]...)
	}
	return strings.Join(out, delim)
}

func slurp(env *fsx.Env, paths []string) (string, int) {
	if len(paths) == 0 {
		if env.Stdin == nil {
			return "", exit.Ok
		}
		b, err := io.ReadAll(env.Stdin)
		if err != nil {
			fmt.Fprintf(env.Stderr, "cut: stdin: %s\n", err)
			return "", exit.Fail
		}
		return string(b), exit.Ok
	}
	var sb strings.Builder
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		b, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "cut: %s: %s\n", p, prettyErr(err))
			return "", exit.Fail
		}
		sb.Write(b)
	}
	return sb.String(), exit.Ok
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
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
