// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package printf

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run formats its operands according to FORMAT. Conversions: %s %d %x %o %c
// %%. Escape sequences: \n \t \\. Excess operands cause the format string to
// be reused (matching GNU printf); excess specs receive zero values.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "printf: missing format")
		return exit.Usage
	}
	format := unescape(args[0])
	operands := args[1:]
	specs := countSpecs(format)
	rc := exit.Ok
	// Always apply the format at least once -- with zero operands this prints
	// zero-valued defaults (matching GNU printf), and lets `%%` collapse to
	// `%` even on a no-spec format.
	first := true
	for first || len(operands) > 0 {
		first = false
		chunk := operands
		if specs > 0 && len(chunk) > specs {
			chunk = chunk[:specs]
		}
		if !render(env, format, chunk) {
			rc = exit.Fail
		}
		if specs == 0 {
			break
		}
		if len(operands) <= specs {
			break
		}
		operands = operands[specs:]
	}
	return rc
}

// unescape collapses the small set of recognised C escapes inside FORMAT.
func unescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case 't':
				b.WriteByte('\t')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// countSpecs returns the number of % conversion specifiers in format (not
// counting %% literals).
func countSpecs(format string) int {
	n := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		if i+1 >= len(format) {
			continue
		}
		if format[i+1] == '%' {
			i++
			continue
		}
		n++
		i++
	}
	return n
}

// render walks format and substitutes each %X with the matching operand,
// drawing on Go's fmt package for the heavy lifting. Returns false on a bad
// conversion (e.g. %d operand that does not parse as int).
func render(env *fsx.Env, format string, operands []string) bool {
	ok := true
	idx := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			fmt.Fprint(env.Stdout, string(format[i]))
			continue
		}
		if i+1 >= len(format) {
			fmt.Fprint(env.Stdout, "%")
			continue
		}
		conv := format[i+1]
		i++
		if conv == '%' {
			fmt.Fprint(env.Stdout, "%")
			continue
		}
		op := ""
		if idx < len(operands) {
			op = operands[idx]
		}
		idx++
		switch conv {
		case 's':
			fmt.Fprint(env.Stdout, op)
		case 'd':
			v, err := strconv.Atoi(op)
			if err != nil && op != "" {
				fmt.Fprintf(env.Stderr, "printf: %q: invalid number\n", op)
				ok = false
				continue
			}
			fmt.Fprintf(env.Stdout, "%d", v)
		case 'x':
			v, err := strconv.Atoi(op)
			if err != nil && op != "" {
				fmt.Fprintf(env.Stderr, "printf: %q: invalid number\n", op)
				ok = false
				continue
			}
			fmt.Fprintf(env.Stdout, "%x", v)
		case 'o':
			v, err := strconv.Atoi(op)
			if err != nil && op != "" {
				fmt.Fprintf(env.Stderr, "printf: %q: invalid number\n", op)
				ok = false
				continue
			}
			fmt.Fprintf(env.Stdout, "%o", v)
		case 'c':
			if len(op) > 0 {
				fmt.Fprint(env.Stdout, string(op[0]))
			}
		default:
			fmt.Fprintf(env.Stderr, "printf: %%%c: unsupported conversion\n", conv)
			ok = false
		}
	}
	return ok
}
