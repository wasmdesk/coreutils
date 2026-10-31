// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package xargs implements the xargs coreutil.
package xargs

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// DefaultSize is the command line size with no -s: 128 KiB, the bytes of
// the arguments, each with its terminating NUL.
const DefaultSize = 128 << 10

// MaxArg bounds one input argument (a line with -I, -L).
const MaxArg = 1 << 20

// Exit statuses besides 0, 126 and 127 (POSIX: 1-125; GNU's values).
const (
	StatusFailed  = 123 // an invocation exited 1-125
	StatusAborted = 124 // an invocation exited 255
)

// Run is POSIX xargs:
//
//	xargs [-rtx] [-E eofstr | -0] [-I replstr | -L number | -n number] [-s size] [utility [argument...]]
//
// It reads arguments from standard input — blanks and newlines separate
// them; ' and " quote, \ escapes, a newline inside quotes is an error;
// with -0 only NUL separates and nothing is special — and runs utility
// (echo by default) with the initial arguments followed by as many as fit
// (-n at most number, -s at most size bytes). -L runs it for each number
// lines (a line ending in a blank continues on the next one). -I runs it
// for each line, its leading blanks dropped, replacing replstr in the
// initial arguments (implies -x). -E stops at an argument equal to eofstr.
// -r does not run the utility when there are no arguments; -t writes each
// command to standard error first; -x stops when one does not fit. The
// utility gets no standard input. Exit: 0, 123 if an invocation failed,
// 124 (and stop) if one exited 255, 126/127 if it cannot run / is not
// found, 1 for a size or input error.
func Run(env *fsx.Env) int {
	o, err := parseArgs(env.Args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "xargs: %v\n", err)
		return exit.Usage
	}
	var in io.Reader = strings.NewReader("")
	if env.Stdin != nil {
		in = env.Stdin
	}
	x := &runner{env: env, o: o, in: newScanner(in, o)}
	return x.run()
}

type options struct {
	cmd        []string // utility and initial arguments
	eof        string
	haveEOF    bool
	nul        bool
	replace    string
	insert     bool
	lines      int // -L
	max        int // -n
	size       int // -s
	noEmpty    bool
	trace      bool
	exitOnSize bool
}

func parseArgs(argv []string) (*options, error) {
	if len(argv) > 0 {
		argv = argv[1:]
	}
	o := &options{size: DefaultSize}
	i := 0
	for ; i < len(argv); i++ {
		a := argv[i]
		if a == "--" {
			i++
			break
		}
		if len(a) < 2 || a[0] != '-' {
			break
		}
		for j := 1; j < len(a); j++ {
			c := a[j]
			switch c {
			case 'r':
				o.noEmpty = true
				continue
			case 't':
				o.trace = true
				continue
			case 'x':
				o.exitOnSize = true
				continue
			case '0':
				o.nul = true
				continue
			case 'p':
				return nil, errors.New("-p is not supported: there is no terminal to ask")
			case 'E', 'I', 'L', 'n', 's':
			default:
				return nil, fmt.Errorf("invalid option -- '%c'", c)
			}
			v := a[j+1:]
			if v == "" {
				i++
				if i >= len(argv) {
					return nil, fmt.Errorf("option requires an argument -- '%c'", c)
				}
				v = argv[i]
			}
			j = len(a)
			switch c {
			case 'E':
				o.eof, o.haveEOF = v, v != ""
			case 'I':
				if v == "" {
					return nil, errors.New("-I: the replacement string is empty")
				}
				o.replace, o.insert, o.lines, o.max = v, true, 0, 0
			default:
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 {
					return nil, fmt.Errorf("-%c: invalid number %q", c, v)
				}
				switch c {
				case 'L':
					o.lines, o.max, o.insert = n, 0, false
				case 'n':
					o.max, o.lines, o.insert = n, 0, false
				case 's':
					o.size = n
				}
			}
		}
	}
	o.cmd = argv[i:]
	if len(o.cmd) == 0 {
		o.cmd = []string{"echo"}
	}
	if o.insert {
		o.exitOnSize = true
	}
	return o, nil
}

// scanner reads the input arguments.
type scanner struct {
	r *bufio.Reader
	o *options
}

func newScanner(r io.Reader, o *options) *scanner { return &scanner{bufio.NewReader(r), o} }

var errQuote = errors.New("unmatched quote")

// next returns the next argument and whether a line ended right after it
// (for -L: a line ending in a blank does not end). io.EOF at the end.
func (s *scanner) next() (string, bool, error) {
	if s.o.nul {
		b, err := s.r.ReadString(0)
		if err != nil && (err != io.EOF || b == "") {
			return "", false, err
		}
		if len(b) > MaxArg {
			return "", false, errTooLong
		}
		return strings.TrimSuffix(b, "\x00"), true, nil
	}
	var b strings.Builder
	have := false
	var quote byte
	for {
		c, err := s.r.ReadByte()
		if err != nil {
			if quote != 0 {
				return "", false, errQuote
			}
			if have {
				return b.String(), true, nil
			}
			return "", false, err
		}
		if b.Len() > MaxArg {
			return "", false, errTooLong
		}
		switch {
		case quote != 0:
			if c == '\n' {
				return "", false, errQuote
			}
			if c == quote {
				quote = 0
				continue
			}
			b.WriteByte(c)
		case c == '\'' || c == '"':
			quote, have = c, true
		case c == '\\':
			e, err := s.r.ReadByte()
			if err != nil {
				return "", false, errors.New("backslash at end of input")
			}
			b.WriteByte(e)
			have = true
		case c == '\n':
			if have {
				return b.String(), true, nil
			}
			// an empty line, or blanks only: nothing
		case c == ' ' || c == '\t':
			switch {
			case s.o.insert && have:
				b.WriteByte(c) // -I: blanks inside a line are part of it
			case have:
				// The argument ends; whether its line ends is known
				// only from what follows on that line.
				return b.String(), s.lineEndsAfterBlanks(), nil
			}
			// leading blanks: dropped
		default:
			b.WriteByte(c)
			have = true
		}
	}
}

var errTooLong = errors.New("argument line too long")

// lineEndsAfterBlanks consumes blanks after an argument and reports
// whether a newline follows them — and then whether the line ends there
// (it does not when blanks preceded the newline: -L continuation).
func (s *scanner) lineEndsAfterBlanks() bool {
	for {
		c, err := s.r.ReadByte()
		if err != nil {
			return true
		}
		if c == ' ' || c == '\t' {
			continue
		}
		if c == '\n' {
			return false // trailing blanks: the line continues (POSIX -L)
		}
		s.r.UnreadByte()
		return false
	}
}

type runner struct {
	env    *fsx.Env
	o      *options
	in     *scanner
	status int
	ran    bool
}

// cmdSize is the size of a command line: each argument and its NUL.
func cmdSize(args []string) int {
	n := 0
	for _, a := range args {
		n += len(a) + 1
	}
	return n
}

func (x *runner) run() int {
	o := x.o
	base := cmdSize(o.cmd)
	if !o.insert && base > o.size {
		fmt.Fprintln(x.env.Stderr, "xargs: the utility and its arguments do not fit in the size")
		return 1
	}
	var batch []string
	size, lines := base, 0
	for {
		arg, lineEnd, err := x.in.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintf(x.env.Stderr, "xargs: %v\n", err)
			x.flush(batch)
			return 1
		}
		if o.haveEOF && arg == o.eof {
			break
		}
		if o.insert {
			cmd := make([]string, len(o.cmd))
			for i, a := range o.cmd {
				cmd[i] = strings.ReplaceAll(a, o.replace, arg)
			}
			if cmdSize(cmd) > o.size {
				fmt.Fprintln(x.env.Stderr, "xargs: argument line too long")
				return 1
			}
			if stop := x.invoke(cmd); stop {
				return x.status
			}
			continue
		}
		if size+len(arg)+1 > o.size {
			if len(batch) == 0 || o.exitOnSize {
				fmt.Fprintln(x.env.Stderr, "xargs: argument line too long")
				x.flush(batch)
				return 1
			}
			if stop := x.flush(batch); stop {
				return x.status
			}
			batch, size, lines = nil, base, 0
		}
		batch = append(batch, arg)
		size += len(arg) + 1
		if lineEnd {
			lines++
		}
		if (o.max > 0 && len(batch) >= o.max) || (o.lines > 0 && lines >= o.lines) {
			if stop := x.flush(batch); stop {
				return x.status
			}
			batch, size, lines = nil, base, 0
		}
	}
	if len(batch) > 0 || (!x.ran && !o.noEmpty && !o.insert) {
		x.flush(batch)
	}
	return x.status
}

// flush runs the utility with batch; true means stop.
func (x *runner) flush(batch []string) bool {
	return x.invoke(append(append([]string(nil), x.o.cmd...), batch...))
}

// invoke runs one command line; true means stop.
func (x *runner) invoke(argv []string) bool {
	x.ran = true
	if x.o.trace {
		fmt.Fprintln(x.env.Stderr, strings.Join(argv, " "))
	}
	code := x.env.RunStatus("xargs", fsx.Command{
		Name: argv[0], Args: argv, Dir: x.env.Cwd,
		Stdout: x.env.Stdout, Stderr: x.env.Stderr,
	})
	switch {
	case code == 0:
	case code == 126 || code == 127:
		x.status = code
		return true
	case code == 255:
		fmt.Fprintf(x.env.Stderr, "xargs: %s: exited with status 255; aborting\n", argv[0])
		x.status = StatusAborted
		return true
	default:
		x.status = StatusFailed
	}
	return false
}
