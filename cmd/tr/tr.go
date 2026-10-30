// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tr

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run is POSIX tr in the POSIX locale (bytes):
//
//	tr [-c|-C] [-s] string1 string2
//	tr -s [-c|-C] string1
//	tr -d [-c|-C] string1
//	tr -ds [-c|-C] string1 string2
//
// Operands: characters, escapes (\\ \a \b \f \n \r \t \v, \ooo octal),
// ranges c-c, classes [:alnum:] [:alpha:] [:blank:] [:cntrl:] [:digit:]
// [:graph:] [:lower:] [:print:] [:punct:] [:space:] [:upper:] [:xdigit:],
// equivalence classes [=c=], and in string2 [x*n] (n copies; a leading 0
// makes n octal) and [x*] (as many as string1 needs). In string2, only
// [:upper:] and [:lower:] are allowed, aligned with [:lower:] and
// [:upper:] in string1 (case conversion). A string2 shorter than string1
// is padded with its last character, as GNU and BSD do. -c (and -C, the
// same in the POSIX locale) complements string1 in ascending byte order.
// -s squeezes runs of a character of the last operand. Input is processed
// as it arrives.
func Run(env *fsx.Env) int {
	o, err := parseArgs(env.Args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tr: %v\n", err)
		return exit.Usage
	}
	if env.Stdin == nil {
		return exit.Ok
	}
	buf := make([]byte, 32<<10)
	out := make([]byte, 0, len(buf))
	last, have := byte(0), false // the last byte written, for -s across reads
	for {
		n, rerr := env.Stdin.Read(buf)
		out = out[:0]
		for _, c := range buf[:n] {
			if o.delete[c] {
				continue
			}
			c = o.xlat[c]
			if o.squeeze[c] && have && c == last {
				continue
			}
			out = append(out, c)
			last, have = c, true
		}
		if len(out) > 0 {
			if _, err := env.Stdout.Write(out); err != nil {
				fmt.Fprintf(env.Stderr, "tr: write error: %v\n", err)
				return exit.Fail
			}
		}
		if rerr != nil {
			if !errors.Is(rerr, io.EOF) {
				fmt.Fprintf(env.Stderr, "tr: stdin: %v\n", rerr)
				return exit.Fail
			}
			return exit.Ok
		}
	}
}

// program is the compiled operation: delete, then translate, then squeeze.
type program struct {
	delete  [256]bool
	xlat    [256]byte
	squeeze [256]bool
}

func parseArgs(argv []string) (*program, error) {
	if len(argv) > 0 {
		argv = argv[1:]
	}
	var comp, del, sq bool
	var ops []string
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "--" {
			ops = append(ops, argv[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			ops = append(ops, argv[i:]...) // operands start here
			break
		}
		for _, c := range a[1:] {
			switch c {
			case 'c', 'C':
				comp = true
			case 'd':
				del = true
			case 's':
				sq = true
			default:
				return nil, fmt.Errorf("invalid option -- '%c'", c)
			}
		}
	}
	want := 2
	switch {
	case del && sq:
	case del || (sq && len(ops) == 1):
		want = 1
	}
	if len(ops) != want {
		return nil, fmt.Errorf("expected %d operand(s), got %d", want, len(ops))
	}
	p := &program{}
	for i := range p.xlat {
		p.xlat[i] = byte(i)
	}
	s1, err := parseSet(ops[0], false)
	if err != nil {
		return nil, err
	}
	a1 := s1.bytes(0)
	if comp {
		a1 = complement(a1)
	}
	switch {
	case del:
		for _, c := range a1 {
			p.delete[c] = true
		}
		if sq {
			s2, err := parseSet(ops[1], false)
			if err != nil {
				return nil, err
			}
			for _, c := range s2.bytes(0) {
				p.squeeze[c] = true
			}
		}
	case len(ops) == 1: // -s string1
		for _, c := range a1 {
			p.squeeze[c] = true
		}
	default:
		s2, err := parseSet(ops[1], true)
		if err != nil {
			return nil, err
		}
		if err := s2.checkCases(s1, comp); err != nil {
			return nil, err
		}
		a2 := s2.bytes(len(a1))
		if len(a2) == 0 && len(a1) > 0 {
			return nil, errors.New("string2 must be non-empty when translating")
		}
		for i, c := range a1 {
			p.xlat[c] = a2[min(i, len(a2)-1)]
		}
		if sq {
			for _, c := range a2 {
				p.squeeze[c] = true
			}
		}
	}
	return p, nil
}

func complement(a []byte) []byte {
	var in [256]bool
	for _, c := range a {
		in[c] = true
	}
	var out []byte
	for c := 0; c < 256; c++ {
		if !in[c] {
			out = append(out, byte(c))
		}
	}
	return out
}

// item is one element of an operand: bytes, a class, or a repetition.
type item struct {
	b     []byte
	class string // "upper"/"lower" when the item is that class
	fill  bool   // [x*]: as many as needed
	pos   int    // index of the item's first byte in the expanded array
}

type set struct{ items []item }

// bytes expands the set; a [x*] fills up to target bytes.
func (s *set) bytes(target int) []byte {
	fixed := 0
	for _, it := range s.items {
		if !it.fill {
			fixed += len(it.b)
		}
	}
	var out []byte
	for i := range s.items {
		it := &s.items[i]
		it.pos = len(out)
		if it.fill {
			out = append(out, strings.Repeat(string(it.b), max(target-fixed, 0))...)
			continue
		}
		out = append(out, it.b...)
	}
	return out
}

// checkCases enforces POSIX's rule for classes in string2: only upper and
// lower, each at the position of the other one in string1.
func (s *set) checkCases(s1 *set, comp bool) error {
	for _, it := range s.items {
		if it.class == "" {
			continue
		}
		other := map[string]string{"upper": "lower", "lower": "upper"}[it.class]
		matched := false
		if !comp {
			for _, j := range s1.items {
				if j.class == other && j.pos == it.pos {
					matched = true
				}
			}
		}
		if !matched {
			return fmt.Errorf("misaligned [:%s:] in string2: it must face [:%s:] in string1", it.class, other)
		}
	}
	return nil
}

var classes = map[string]func(c byte) bool{
	"alnum":  func(c byte) bool { return isAlpha(c) || isDigit(c) },
	"alpha":  isAlpha,
	"blank":  func(c byte) bool { return c == ' ' || c == '\t' },
	"cntrl":  func(c byte) bool { return c < 0x20 || c == 0x7f },
	"digit":  isDigit,
	"graph":  func(c byte) bool { return c > 0x20 && c < 0x7f },
	"lower":  func(c byte) bool { return c >= 'a' && c <= 'z' },
	"print":  func(c byte) bool { return c >= 0x20 && c < 0x7f },
	"punct":  func(c byte) bool { return c > 0x20 && c < 0x7f && !isAlpha(c) && !isDigit(c) },
	"space":  func(c byte) bool { return c == ' ' || (c >= '\t' && c <= '\r') },
	"upper":  func(c byte) bool { return c >= 'A' && c <= 'Z' },
	"xdigit": func(c byte) bool { return isDigit(c) || (c|0x20) >= 'a' && (c|0x20) <= 'f' },
}

func isAlpha(c byte) bool { return (c|0x20) >= 'a' && (c|0x20) <= 'z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// parseSet reads an operand; repetitions are allowed in string2 only.
func parseSet(s string, second bool) (*set, error) {
	st := &set{}
	i := 0
	// char reads one (possibly escaped) character at i.
	char := func() (byte, error) {
		c := s[i]
		i++
		if c != '\\' || i >= len(s) {
			return c, nil // a lone trailing backslash is itself
		}
		e := s[i]
		i++
		switch e {
		case '\\':
			return '\\', nil
		case 'a':
			return '\a', nil
		case 'b':
			return '\b', nil
		case 'f':
			return '\f', nil
		case 'n':
			return '\n', nil
		case 'r':
			return '\r', nil
		case 't':
			return '\t', nil
		case 'v':
			return '\v', nil
		}
		if e >= '0' && e <= '7' {
			j := i - 1
			for i < len(s) && i-j < 3 && s[i] >= '0' && s[i] <= '7' {
				i++
			}
			v, _ := strconv.ParseUint(s[j:i], 8, 16)
			if v > 0xff {
				return 0, fmt.Errorf("octal escape \\%s is not a byte", s[j:i])
			}
			return byte(v), nil
		}
		return e, nil // an unknown escape: the character
	}
	for i < len(s) {
		if s[i] == '[' && i+1 < len(s) {
			rest := s[i+1:]
			switch {
			case strings.HasPrefix(rest, ":"):
				if j := strings.Index(rest[1:], ":]"); j >= 0 {
					name := rest[1 : 1+j]
					is, ok := classes[name]
					if !ok {
						return nil, fmt.Errorf("invalid character class %q", name)
					}
					if second && name != "upper" && name != "lower" {
						return nil, errors.New("the only character classes that may appear in string2 are 'upper' and 'lower'")
					}
					var b []byte
					for c := 0; c < 256; c++ {
						if is(byte(c)) {
							b = append(b, byte(c))
						}
					}
					it := item{b: b}
					if name == "upper" || name == "lower" {
						it.class = name
					}
					st.items = append(st.items, it)
					i += j + 4
					continue
				}
			case strings.HasPrefix(rest, "="):
				if len(rest) >= 4 && rest[2:4] == "=]" { // [=c=]: c itself in the POSIX locale
					st.items = append(st.items, item{b: []byte{rest[1]}})
					i += 5
					continue
				}
			default:
				// [x*n] or [x*]
				save := i
				i++
				x, err := char()
				if err == nil && i < len(s) && s[i] == '*' {
					j := strings.IndexByte(s[i:], ']')
					if j >= 0 {
						if !second {
							return nil, errors.New("[c*] repetition is only allowed in string2")
						}
						num := s[i+1 : i+j]
						i += j + 1
						if num == "" {
							st.items = append(st.items, item{b: []byte{x}, fill: true})
							continue
						}
						base := 10
						if num[0] == '0' {
							base = 8
						}
						n, err := strconv.ParseUint(num, base, 31)
						if err != nil {
							return nil, fmt.Errorf("invalid repeat count %q", num)
						}
						if n == 0 {
							st.items = append(st.items, item{b: []byte{x}, fill: true})
							continue
						}
						if n > 1<<20 {
							return nil, fmt.Errorf("repeat count %q is too large", num)
						}
						st.items = append(st.items, item{b: []byte(strings.Repeat(string(x), int(n)))})
						continue
					}
				}
				i = save
			}
		}
		lo, err := char()
		if err != nil {
			return nil, err
		}
		if i+1 < len(s) && s[i] == '-' {
			i++
			hi, err := char()
			if err != nil {
				return nil, err
			}
			if hi < lo {
				return nil, fmt.Errorf("range-endpoints of '%c-%c' are in reverse order", lo, hi)
			}
			var b []byte
			for c := int(lo); c <= int(hi); c++ {
				b = append(b, byte(c))
			}
			st.items = append(st.items, item{b: b})
			continue
		}
		st.items = append(st.items, item{b: []byte{lo}})
	}
	return st, nil
}
