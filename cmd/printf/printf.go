// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package printf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// MaxWidth bounds a field width or precision: a pad, not a buffer size.
const MaxWidth = 1 << 16

// Run is POSIX printf: printf format [argument...].
//
// Format escapes: \\ \a \b \f \n \r \t \v \ddd (octal). Conversions:
// d i o u x X (integers: C constants, a leading sign, 'c for a character's
// value), f F e E g G a A (floating point), c, s, b (escapes in the
// argument, \0ddd, \c stops all output), %%. Flags - + space # 0 (and ',
// accepted: no grouping in the POSIX locale), a width and a precision,
// each a number or * (taken from the arguments), and %n$ positional
// arguments. The format is reused while arguments remain; a missing one
// is an empty string or zero. A bad number is reported, what was parsed of
// it is used, and the exit status is 1.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "printf: missing format")
		return exit.Usage
	}
	p := &printer{env: env, format: args[0], args: args[1:]}
	var out bytes.Buffer
	p.out = &out
	for {
		used, stop := p.pass()
		if stop || used == 0 || len(p.args) <= used {
			break
		}
		p.args = p.args[used:]
	}
	if _, err := env.Stdout.Write(out.Bytes()); err != nil {
		fmt.Fprintf(env.Stderr, "printf: write error: %v\n", err)
		return exit.Fail
	}
	return p.rc
}

type printer struct {
	env    *fsx.Env
	format string
	args   []string
	out    io.Writer
	rc     int
	next   int // the next unnumbered argument in this pass
	maxPos int // the highest %n$ used in this pass
}

// arg returns argument i (0-based) of this pass, and whether it exists.
func (p *printer) arg(i int) (string, bool) {
	if i < len(p.args) {
		return p.args[i], true
	}
	return "", false
}

// pass applies the format once; it returns how many arguments it used
// and whether \c stopped everything.
func (p *printer) pass() (int, bool) {
	p.next, p.maxPos = 0, 0
	f := p.format
	for i := 0; i < len(f); i++ {
		c := f[i]
		switch c {
		case '\\':
			n, s := escape(f[i+1:], false)
			io.WriteString(p.out, s)
			i += n
			continue
		case '%':
		default:
			p.out.Write([]byte{c})
			continue
		}
		n, stop := p.conversion(f[i+1:])
		i += n
		if stop {
			return 0, true
		}
	}
	return max(p.next, p.maxPos), false
}

// conversion handles one specification (after its %); it returns the
// bytes it consumed and whether \c (in %b) stopped the output.
func (p *printer) conversion(f string) (int, bool) {
	i := 0
	if i < len(f) && f[i] == '%' {
		io.WriteString(p.out, "%")
		return 1, false
	}
	// %n$
	pos := -1
	if j := strings.IndexByte(f, '$'); j > 0 && allDigits(f[:j]) {
		n, _ := strconv.Atoi(f[:j])
		if n >= 1 {
			pos, i = n-1, j+1
		}
	}
	var flags []byte
	for i < len(f) && strings.IndexByte("-+ #0'", f[i]) >= 0 {
		if f[i] != '\'' {
			flags = append(flags, f[i])
		}
		i++
	}
	width, precision := -1, -1
	readNum := func() int {
		if i < len(f) && f[i] == '*' {
			i++
			v, _ := p.intArg(p.take(-1))
			return int(max(min(v, MaxWidth), -MaxWidth))
		}
		j := i
		for i < len(f) && f[i] >= '0' && f[i] <= '9' {
			i++
		}
		if j == i {
			return -1
		}
		v, _ := strconv.Atoi(f[j:i])
		if v > MaxWidth || v < 0 {
			v = MaxWidth
		}
		return v
	}
	width = readNum()
	if width < -1 { // a negative * width means the - flag
		flags, width = append(flags, '-'), -width
	}
	if i < len(f) && f[i] == '.' {
		i++
		if precision = readNum(); precision < 0 {
			precision = 0
		}
	}
	for i < len(f) && strings.IndexByte("hljztL", f[i]) >= 0 { // length modifiers: ignored
		i++
	}
	if i >= len(f) { // cut short: fatal, as GNU and BSD printf
		fmt.Fprintf(p.env.Stderr, "printf: %%%s: invalid conversion specification\n", f)
		p.rc = exit.Fail
		return len(f), true
	}
	verb := f[i]
	i++
	spec := mkSpec(flags, width, precision)
	textSpec := mkSpec(bytes.ReplaceAll(flags, []byte("0"), nil), width, precision) // no zero padding for text
	switch verb {
	case 'd', 'i':
		v, _ := p.intArg(p.take(pos))
		fmt.Fprintf(p.out, spec+"d", v)
	case 'o', 'u', 'x', 'X':
		v, _ := p.intArg(p.take(pos))
		gv := map[byte]string{'o': "o", 'u': "d", 'x': "x", 'X': "X"}[verb]
		fmt.Fprintf(p.out, spec+gv, uint64(v)) // C converts to unsigned
	case 'f', 'F', 'e', 'E', 'g', 'G', 'a', 'A':
		p.float(spec, flags, width, precision, verb, p.take(pos))
	case 'c':
		s := p.take(pos)
		if s != "" {
			s = s[:1]
		}
		fmt.Fprintf(p.out, mkSpec(bytes.ReplaceAll(flags, []byte("0"), nil), width, -1)+"s", s)
	case 's':
		fmt.Fprintf(p.out, textSpec+"s", p.take(pos))
	case 'b':
		s, stop := expandB(p.take(pos))
		fmt.Fprintf(p.out, textSpec+"s", s)
		if stop {
			return i, true
		}
	default: // fatal, as GNU and BSD printf
		fmt.Fprintf(p.env.Stderr, "printf: %%%c: invalid conversion specification\n", verb)
		p.rc = exit.Fail
		return i, true
	}
	return i, false
}

// mkSpec builds a Go verb prefix: % flags [width] [.precision].
func mkSpec(flags []byte, width, precision int) string {
	spec := "%" + string(flags)
	if width >= 0 {
		spec += strconv.Itoa(width)
	}
	if precision >= 0 {
		spec += "." + strconv.Itoa(precision)
	}
	return spec
}

// take returns argument pos (or the next one for -1), or "" if missing.
func (p *printer) take(pos int) string {
	if pos >= 0 {
		p.maxPos = max(p.maxPos, pos+1)
		s, _ := p.arg(pos)
		return s
	}
	s, _ := p.arg(p.next)
	p.next++
	return s
}

func (p *printer) bad(s, why string) {
	fmt.Fprintf(p.env.Stderr, "printf: %q: %s\n", s, why)
	p.rc = exit.Fail
}

// intArg reads an integer argument as C's strtoimax would (base 0: 0x
// hex, 0 octal), or 'c / "c for c's value; "" is 0.
func (p *printer) intArg(s string) (int64, bool) {
	if s == "" {
		return 0, true
	}
	if s[0] == '\'' || s[0] == '"' {
		if len(s) == 1 {
			return 0, true
		}
		r, _ := utf8.DecodeRuneInString(s[1:])
		return int64(r), true
	}
	t := strings.TrimLeft(s, " \t")
	neg := false
	if t != "" && (t[0] == '+' || t[0] == '-') {
		neg, t = t[0] == '-', t[1:]
	}
	base, digits := 10, t
	switch {
	case len(t) > 1 && (t[:2] == "0x" || t[:2] == "0X"):
		base, digits = 16, t[2:]
	case len(t) > 1 && t[0] == '0':
		base, digits = 8, t[1:]
	}
	end := 0
	for end < len(digits) && digitValue(digits[end]) < base {
		end++
	}
	var v uint64
	var err error
	if end > 0 {
		v, err = strconv.ParseUint(digits[:end], base, 64)
	}
	var r int64
	switch {
	case err != nil || (!neg && v > math.MaxInt64) || (neg && v > 1<<63):
		p.bad(s, "Result too large")
		if neg {
			return math.MinInt64, false
		}
		return math.MaxInt64, false
	case neg:
		r = -int64(v)
	default:
		r = int64(v)
	}
	// "0" alone is octal with no further digits; "0x" alone converts the 0.
	if end != len(digits) || (end == 0 && base != 8) {
		if end == 0 && base == 10 {
			p.bad(s, "expected a numeric value")
		} else {
			p.bad(s, "value not completely converted")
		}
		return r, false
	}
	return r, true
}

func digitValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 99
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// float formats a floating conversion the way C's printf does.
func (p *printer) float(spec string, flags []byte, width, precision int, verb byte, s string) {
	v := 0.0
	if s != "" {
		if s[0] == '\'' || s[0] == '"' {
			if len(s) > 1 {
				r, _ := utf8.DecodeRuneInString(s[1:])
				v = float64(r)
			}
		} else {
			var err error
			v, err = strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				var ne *strconv.NumError
				if errors.As(err, &ne) && errors.Is(ne.Err, strconv.ErrRange) {
					p.bad(s, "Result too large") // v is ±Inf or 0, as strtod
				} else {
					p.bad(s, "expected a numeric value")
					v = 0
				}
			}
		}
	}
	upper := verb == 'F' || verb == 'E' || verb == 'G' || verb == 'A'
	if math.IsInf(v, 0) || math.IsNaN(v) {
		t := "inf"
		if math.IsNaN(v) {
			t = "nan"
		}
		if upper {
			t = strings.ToUpper(t)
		}
		if math.IsInf(v, -1) {
			t = "-" + t
		} else if bytes.IndexByte(flags, '+') >= 0 {
			t = "+" + t
		} else if bytes.IndexByte(flags, ' ') >= 0 {
			t = " " + t
		}
		w := ""
		if width >= 0 {
			w = strconv.Itoa(width)
		}
		minus := ""
		if bytes.IndexByte(flags, '-') >= 0 {
			minus = "-"
		}
		fmt.Fprintf(p.out, "%"+minus+w+"s", t)
		return
	}
	switch verb {
	case 'g', 'G':
		if precision < 0 {
			spec += ".6" // C's default; Go's %g would print the shortest form
		}
		fmt.Fprintf(p.out, spec+string(verb), v)
	case 'a', 'A':
		gv := "x"
		if verb == 'A' {
			gv = "X"
		}
		// Formatted unpadded, the exponent shortened as C writes it, then
		// padded to the width.
		noPad := bytes.ReplaceAll(bytes.ReplaceAll(flags, []byte("0"), nil), []byte("-"), nil)
		s := shortExponent(fmt.Sprintf(mkSpec(noPad, -1, precision)+gv, v))
		fmt.Fprint(p.out, pad(s, width, flags))
	default:
		fmt.Fprintf(p.out, spec+string(verb), v)
	}
}

// shortExponent drops the leading zeros Go writes in a hexadecimal float's
// exponent ("p+00" -> "p+0"), as C does.
func shortExponent(s string) string {
	i := strings.LastIndexAny(s, "pP")
	exp := strings.TrimLeft(s[i+2:], "0")
	if exp == "" {
		exp = "0"
	}
	return s[:i+2] + exp
}

// pad pads a formatted number to width: on the right with '-', with zeros
// after the sign and 0x prefix with '0', else with spaces on the left.
func pad(s string, width int, flags []byte) string {
	n := width - len(s)
	if n <= 0 {
		return s
	}
	switch {
	case bytes.IndexByte(flags, '-') >= 0:
		return s + strings.Repeat(" ", n)
	case bytes.IndexByte(flags, '0') >= 0:
		k := 0
		if k < len(s) && (s[k] == '+' || s[k] == '-' || s[k] == ' ') {
			k++
		}
		if k+1 < len(s) && s[k] == '0' && (s[k+1] == 'x' || s[k+1] == 'X') {
			k += 2
		}
		return s[:k] + strings.Repeat("0", n) + s[k:]
	}
	return strings.Repeat(" ", n) + s
}

// escape decodes one backslash escape at the start of s (after the
// backslash); inB selects %b's rules (\0ddd, \c). It returns the bytes
// consumed and the text. A lone or unknown escape is kept as written.
func escape(s string, inB bool) (int, string) {
	if s == "" {
		return 0, "\\"
	}
	switch c := s[0]; c {
	case '\\':
		return 1, "\\"
	case 'a':
		return 1, "\a"
	case 'b':
		return 1, "\b"
	case 'f':
		return 1, "\f"
	case 'n':
		return 1, "\n"
	case 'r':
		return 1, "\r"
	case 't':
		return 1, "\t"
	case 'v':
		return 1, "\v"
	case '"':
		return 1, "\""
	}
	if s[0] >= '0' && s[0] <= '7' {
		start, limit := 0, 3
		if inB && s[0] == '0' { // \0ddd: up to three digits after the 0
			start, limit = 1, 4
		}
		j := start
		for j < len(s) && j < limit && s[j] >= '0' && s[j] <= '7' {
			j++
		}
		v, _ := strconv.ParseUint("0"+s[start:j], 8, 16)
		return j, string([]byte{byte(v)})
	}
	return 1, "\\" + s[:1]
}

// expandB expands %b's argument; true when \c stopped the output.
func expandB(s string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 < len(s) && s[i+1] == 'c' {
			return b.String(), true
		}
		n, t := escape(s[i+1:], true)
		b.WriteString(t)
		i += n
	}
	return b.String(), false
}
