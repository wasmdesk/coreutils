// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package sort

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run is POSIX sort (XCU sort) in the POSIX locale (bytes):
//
//	sort [-bdfinru] [-t char] [-k keydef]... [-o output] [file...]
//
// A keydef is field_start[type][,field_end[type]], each position being
// field[.char] and type some of b d f i n r. A key with a modifier ignores
// the global options. Lines whose keys compare equal are ordered by the
// whole line, byte by byte, unless -u, which keeps the first of each run
// of equal keys. -c checks the order instead of sorting.
func Run(env *fsx.Env) int {
	o, paths, err := parseArgs(env.Args)
	if err != nil {
		fmt.Fprintf(env.Stderr, "sort: %v\n", err)
		return exit.Usage
	}
	lines, rc := collect(env, paths)
	if rc != exit.Ok {
		return rc
	}
	cmp := o.compare
	if o.check {
		for i := 1; i < len(lines); i++ {
			if c := cmp(lines[i-1], lines[i]); c > 0 || (o.unique && c == 0) {
				fmt.Fprintf(env.Stderr, "sort: disorder: %s\n", lines[i])
				return exit.Fail
			}
		}
		return exit.Ok
	}
	o.sortLines(lines)
	if o.unique {
		lines = dedup(lines, cmp)
	}
	var out io.Writer = env.Stdout
	if o.output != "" {
		// Opened after reading, so -o may name an input file.
		w, err := fsx.Create(env.FS, fsx.Resolve(env.Cwd, o.output), false)
		if err != nil {
			fmt.Fprintf(env.Stderr, "sort: %s: %s\n", o.output, prettyErr(err))
			return exit.Fail
		}
		defer w.Close()
		out = w
	}
	var b bytes.Buffer
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	if _, err := out.Write(b.Bytes()); err != nil {
		fmt.Fprintf(env.Stderr, "sort: write error: %v\n", err)
		return exit.Fail
	}
	return exit.Ok
}

// mods are the ordering modifiers of a key (or the global ones).
type mods struct{ b, d, f, i, n, r bool }

func (m mods) any() bool { return m.b || m.d || m.f || m.i || m.n || m.r }

type pos struct{ field, char int }

type key struct {
	start, end   pos // end.field 0: end of line; end.char 0: end of field
	m            mods
	bStart, bEnd bool
}

type options struct {
	global mods
	keys   []key
	sep    int // -t byte, or -1
	unique bool
	check  bool
	output string
}

func parseArgs(argv []string) (*options, []string, error) {
	o := &options{sep: -1}
	if len(argv) > 0 {
		argv = argv[1:]
	}
	var paths []string
	var keydefs []string
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
			c := a[j]
			switch c {
			case 'b', 'd', 'f', 'i', 'n', 'r':
				setMod(&o.global, c)
			case 'u':
				o.unique = true
			case 'c', 'C':
				o.check = true
			case 'k', 't', 'o':
				v := a[j+1:]
				if v == "" {
					i++
					if i >= len(argv) {
						return nil, nil, fmt.Errorf("option requires an argument -- '%c'", c)
					}
					v = argv[i]
				}
				switch c {
				case 'k':
					keydefs = append(keydefs, v)
				case 't':
					if len(v) != 1 {
						return nil, nil, fmt.Errorf("-t: the separator must be one character: %q", v)
					}
					o.sep = int(v[0])
				case 'o':
					o.output = v
				}
				j = len(a)
			default:
				return nil, nil, fmt.Errorf("invalid option -- '%c'", c)
			}
		}
	}
	for _, kd := range keydefs {
		k, err := parseKey(kd, o.global)
		if err != nil {
			return nil, nil, err
		}
		o.keys = append(o.keys, k)
	}
	return o, paths, nil
}

func setMod(m *mods, c byte) {
	switch c {
	case 'b':
		m.b = true
	case 'd':
		m.d = true
	case 'f':
		m.f = true
	case 'i':
		m.i = true
	case 'n':
		m.n = true
	case 'r':
		m.r = true
	}
}

// parseKey reads field_start[type][,field_end[type]].
func parseKey(kd string, global mods) (key, error) {
	bad := func() (key, error) { return key{}, fmt.Errorf("-k: invalid key %q", kd) }
	startS, endS, hasEnd := strings.Cut(kd, ",")
	var k key
	var sm, em mods
	var err error
	if k.start, sm, err = parsePos(startS, true); err != nil {
		return bad()
	}
	if hasEnd {
		if k.end, em, err = parsePos(endS, false); err != nil {
			return bad()
		}
	}
	k.m = mods{d: sm.d || em.d, f: sm.f || em.f, i: sm.i || em.i, n: sm.n || em.n, r: sm.r || em.r}
	k.bStart, k.bEnd = sm.b, em.b
	if !sm.any() && !em.any() {
		// No modifier: the global options apply (-b to both ends).
		k.m = global
		k.bStart, k.bEnd = global.b, global.b
	}
	return k, nil
}

func parsePos(s string, start bool) (pos, mods, error) {
	var m mods
	i := len(s)
	for i > 0 && strings.IndexByte("bdfinr", s[i-1]) >= 0 {
		setMod(&m, s[i-1])
		i--
	}
	fs, cs, hasChar := strings.Cut(s[:i], ".")
	f, err := strconv.Atoi(fs)
	if err != nil || f < 1 {
		return pos{}, m, errors.New("bad field")
	}
	p := pos{field: f}
	if hasChar {
		c, err := strconv.Atoi(cs)
		if err != nil || c < 0 || (start && c < 1) {
			return pos{}, m, errors.New("bad character")
		}
		p.char = c
	}
	return p, m, nil
}

func isBlank(c byte) bool { return c == ' ' || c == '\t' }

// fieldStart returns the offset where field f (1-based) begins: with -t,
// after f-1 separators; without, before the blanks that precede it.
func (o *options) fieldStart(l string, f int) int {
	i := 0
	for n := 1; n < f && i < len(l); n++ {
		if o.sep >= 0 {
			j := strings.IndexByte(l[i:], byte(o.sep))
			if j < 0 {
				return len(l)
			}
			i += j + 1
			continue
		}
		for i < len(l) && isBlank(l[i]) {
			i++
		}
		for i < len(l) && !isBlank(l[i]) {
			i++
		}
	}
	return i
}

// fieldEnd returns the offset just past field f, starting at its start.
func (o *options) fieldEnd(l string, i int) int {
	if o.sep >= 0 {
		if j := strings.IndexByte(l[i:], byte(o.sep)); j >= 0 {
			return i + j
		}
		return len(l)
	}
	for i < len(l) && isBlank(l[i]) {
		i++
	}
	for i < len(l) && !isBlank(l[i]) {
		i++
	}
	return i
}

func skipBlanks(l string, i int) int {
	for i < len(l) && isBlank(l[i]) {
		i++
	}
	return i
}

// extract returns the part of l that key k compares.
func (o *options) extract(l string, k key) string {
	s := o.fieldStart(l, k.start.field)
	if k.bStart {
		s = skipBlanks(l, s)
	}
	if k.start.char > 0 {
		s = min(s+k.start.char-1, len(l))
	}
	e := len(l)
	if k.end.field > 0 {
		fs := o.fieldStart(l, k.end.field)
		if k.end.char == 0 {
			e = o.fieldEnd(l, fs)
		} else {
			if k.bEnd {
				fs = skipBlanks(l, fs)
			}
			e = min(fs+k.end.char, len(l))
		}
	}
	if e < s {
		return ""
	}
	return l[s:e]
}

// record is a line with its keys extracted once (filtered, or parsed as
// numbers for n keys), not at every comparison.
type record struct {
	line string
	keys []string
	nums []numeric
}

func (o *options) sortKeys() []key {
	if len(o.keys) == 0 {
		return []key{{start: pos{field: 1}, m: o.global, bStart: o.global.b, bEnd: o.global.b}}
	}
	return o.keys
}

func (o *options) decorate(l string, keys []key) record {
	return o.decorateInto(l, keys, make([]string, len(keys)), make([]numeric, len(keys)))
}

func (o *options) decorateInto(l string, keys []key, ks []string, ns []numeric) record {
	r := record{line: l, keys: ks, nums: ns}
	for i, k := range keys {
		s := o.extract(l, k)
		if k.m.n {
			r.nums[i] = parseNum(s)
		} else {
			r.keys[i] = filter(s, k.m)
		}
	}
	return r
}

// compare orders two lines by the keys, then (unless -u) by the line.
func (o *options) compare(a, b string) int {
	keys := o.sortKeys()
	x, y := o.decorate(a, keys), o.decorate(b, keys)
	return o.compareRec(&x, &y, keys)
}

func (o *options) compareRec(a, b *record, keys []key) int {
	for i, k := range keys {
		var c int
		if k.m.n {
			c = compareNumeric(a.nums[i], b.nums[i])
		} else {
			c = strings.Compare(a.keys[i], b.keys[i])
		}
		if k.m.r {
			c = -c
		}
		if c != 0 {
			return c
		}
	}
	if o.unique {
		return 0
	}
	c := strings.Compare(a.line, b.line)
	if o.global.r {
		c = -c
	}
	return c
}

// sortLines sorts stably, extracting each line's keys once (into one flat
// array) and sorting indices with pdqsort: the input position breaks ties,
// which makes it stable without a merge sort's rotations.
func (o *options) sortLines(lines []string) {
	keys := o.sortKeys()
	n, k := len(lines), len(keys)
	recs := make([]record, n)
	ks, ns := make([]string, n*k), make([]numeric, n*k)
	idx := make([]int32, n)
	for i, l := range lines {
		recs[i] = o.decorateInto(l, keys, ks[i*k:(i+1)*k:(i+1)*k], ns[i*k:(i+1)*k:(i+1)*k])
		idx[i] = int32(i)
	}
	slices.SortFunc(idx, func(a, b int32) int {
		if c := o.compareRec(&recs[a], &recs[b], keys); c != 0 {
			return c
		}
		return int(a) - int(b)
	})
	for i, j := range idx {
		lines[i] = recs[j].line
	}
}

// filter applies -d (blanks and alphanumerics only), -i (printable only)
// and -f (fold lower case to upper case).
func filter(s string, m mods) string {
	if !m.d && !m.i && !m.f {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if m.d && !(isBlank(c) || isAlnum(c)) {
			continue
		}
		if m.i && (c < 0x20 || c > 0x7e) {
			continue
		}
		if m.f && c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// numeric is the initial numeric string of s: optional blanks, an
// optional '-', digits, an optional '.' and digits. An empty digit string
// is zero; leading zeros and the sign of zero do not matter.
type numeric struct {
	neg       bool
	int, frac string // no leading (int) or trailing (frac) zeros
}

func parseNum(s string) numeric {
	i := skipBlanks(s, 0)
	var n numeric
	if i < len(s) && s[i] == '-' {
		n.neg = true
		i++
	}
	j := i
	for j < len(s) && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	n.int = strings.TrimLeft(s[i:j], "0")
	if j < len(s) && s[j] == '.' {
		k := j + 1
		for k < len(s) && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		n.frac = strings.TrimRight(s[j+1:k], "0")
	}
	if n.int == "" && n.frac == "" {
		n.neg = false // -0 is 0
	}
	return n
}

// compareNum compares exactly, digit by digit (no float rounding).
func compareNum(a, b string) int { return compareNumeric(parseNum(a), parseNum(b)) }

func compareNumeric(x, y numeric) int {
	if x.neg != y.neg {
		if x.neg {
			return -1
		}
		return 1
	}
	c := len(x.int) - len(y.int)
	if c == 0 {
		c = strings.Compare(x.int, y.int)
	}
	if c == 0 {
		c = strings.Compare(x.frac, y.frac)
	}
	if c < 0 {
		c = -1
	} else if c > 0 {
		c = 1
	}
	if x.neg {
		c = -c
	}
	return c
}

// collect reads every input source into a flat slice of newline-stripped
// lines. A final line without a newline is a line.
func collect(env *fsx.Env, paths []string) ([]string, int) {
	var lines []string
	if len(paths) == 0 {
		paths = []string{fsx.StdinOperand}
	}
	for _, p := range paths {
		var data []byte
		var err error
		if p == fsx.StdinOperand {
			if env.Stdin == nil {
				continue
			}
			data, err = io.ReadAll(env.Stdin)
		} else {
			data, err = env.FS.ReadFile(fsx.Resolve(env.Cwd, p))
		}
		if err != nil {
			name := p
			if p == fsx.StdinOperand {
				name = "stdin"
			}
			fmt.Fprintf(env.Stderr, "sort: %s: %s\n", name, prettyErr(err))
			return nil, exit.Fail
		}
		lines = append(lines, splitLines(string(data))...)
	}
	return lines, exit.Ok
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// dedup keeps the first of each run of lines whose keys compare equal.
func dedup(in []string, cmp func(a, b string) int) []string {
	if len(in) == 0 {
		return in
	}
	out := in[:1]
	for _, s := range in[1:] {
		if cmp(out[len(out)-1], s) != 0 {
			out = append(out, s)
		}
	}
	return out
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
