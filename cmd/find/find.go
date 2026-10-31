// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package find

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// MaxDepth bounds the walk: a symbolic link to an ancestor (OSFS follows
// links) cannot recurse forever.
const MaxDepth = 256

// Now is the initialization time -mtime counts from (a seam for tests).
var Now = time.Now

// Run is POSIX find (XCU find): find [path...] [expression]. The paths
// printed are formed from each path operand as given. Primaries: -name,
// -iname, -type (f and d; b c l p s never match: fsx has no such files),
// -size n[cbwkMG], -mtime n, -newer file, -print, -prune, -true,
// -false; operators ( ), ! / -not, -a / -and (or juxtaposition), -o / -or.
// Numbers are n, +n (more than), -n (less than). GNU's global options
// -maxdepth and -mindepth are accepted anywhere in the expression.
// -exec utility [argument...] ; runs utility for each path ({} in an
// argument replaced by it; true if it exits 0); -exec utility
// [argument...] {} + runs it with as many paths as fit (always true; a
// failure makes find's status 1). With no -print or -exec in the
// expression, a path is printed when the expression is true.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var roots []string
	for len(args) > 0 && !isExprStart(args[0]) {
		roots = append(roots, args[0])
		args = args[1:]
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}
	p := &parser{env: env, toks: args, maxDepth: -1}
	expr, err := p.parse()
	if err != nil {
		fmt.Fprintf(env.Stderr, "find: %v\n", err)
		return exit.Usage
	}
	if !p.hasPrint {
		expr = and{expr, printer{}}
	}
	w := &walker{env: env, expr: expr, min: p.minDepth, max: p.maxDepth}
	for _, r := range roots {
		w.walk(r, fsx.Resolve(env.Cwd, r), 0)
	}
	for _, b := range p.batches {
		b.flush(w)
	}
	return w.rc
}

// MaxExecSize bounds the arguments of one -exec ... {} + command line, in
// bytes, each with its NUL (like xargs' default).
const MaxExecSize = 128 << 10

// execOne is -exec ... ;: one run per path.
type execOne struct{ argv []string }

func (x execOne) eval(w *walker, e *entry) bool {
	argv := make([]string, len(x.argv))
	for i, a := range x.argv {
		argv[i] = strings.ReplaceAll(a, "{}", e.shown) // GNU: also inside an argument
	}
	code := w.run(argv)
	return code == 0
}

// execBatch is -exec ... {} +: paths gathered, run in batches.
type execBatch struct {
	argv  []string // without the {}
	paths []string
	size  int
}

func (x *execBatch) eval(w *walker, e *entry) bool {
	n := len(e.shown) + 1
	if len(x.paths) > 0 && x.size+n > MaxExecSize {
		x.flush(w)
	}
	x.paths = append(x.paths, e.shown)
	x.size += n
	return true
}

func (x *execBatch) flush(w *walker) {
	if len(x.paths) == 0 {
		return
	}
	argv := append(append([]string(nil), x.argv...), x.paths...)
	x.paths, x.size = nil, 0
	if w.run(argv) != 0 {
		w.rc = exit.Fail // POSIX: a failing + command makes find's status non-zero
	}
}

// run runs argv with find's stdio; 126/127 make find's status 1.
func (w *walker) run(argv []string) int {
	code := w.env.RunStatus("find", fsx.Command{
		Name: argv[0], Args: argv, Dir: w.env.Cwd,
		Stdin: w.env.Stdin, Stdout: w.env.Stdout, Stderr: w.env.Stderr,
	})
	if code == 126 || code == 127 {
		w.rc = exit.Fail
	}
	return code
}

func isExprStart(a string) bool {
	return a == "!" || a == "(" || (len(a) > 1 && a[0] == '-')
}

// entry is the file an expression is evaluated on.
type entry struct {
	shown string // the path as printed
	info  fsx.FileInfo
	prune bool // set by -prune
}

type expr interface {
	eval(w *walker, e *entry) bool
}

type (
	and      [2]expr
	or       [2]expr
	not      struct{ x expr }
	printer  struct{}
	pruner   struct{}
	constant bool
	name     struct {
		pat  string
		fold bool
	}
	ftype byte
	size  struct{ cmp, n, unit int64 }
	mtime struct{ cmp, n int64 }
	newer struct{ t time.Time }
)

func (x and) eval(w *walker, e *entry) bool { return x[0].eval(w, e) && x[1].eval(w, e) }
func (x or) eval(w *walker, e *entry) bool  { return x[0].eval(w, e) || x[1].eval(w, e) }
func (x not) eval(w *walker, e *entry) bool { return !x.x.eval(w, e) }
func (printer) eval(w *walker, e *entry) bool {
	fmt.Fprintln(w.env.Stdout, e.shown)
	return true
}
func (pruner) eval(_ *walker, e *entry) bool { e.prune = true; return true }
func (x constant) eval(*walker, *entry) bool { return bool(x) }

func (x name) eval(_ *walker, e *entry) bool {
	base, pat := path.Base(e.shown), x.pat
	if x.fold {
		base, pat = strings.ToLower(base), strings.ToLower(pat)
	}
	ok, _ := path.Match(pat, base)
	return ok
}

func (x ftype) eval(_ *walker, e *entry) bool {
	switch x {
	case 'd':
		return e.info.IsDir
	case 'f':
		return !e.info.IsDir
	}
	return false
}

// compare applies the n / +n / -n rule.
func compare(cmp, v, n int64) bool {
	switch {
	case cmp > 0:
		return v > n
	case cmp < 0:
		return v < n
	}
	return v == n
}

func (x size) eval(_ *walker, e *entry) bool {
	// The size in units, rounded up (POSIX: 512-byte blocks).
	v := (e.info.Size + x.unit - 1) / x.unit
	return compare(x.cmp, v, x.n)
}

func (x mtime) eval(w *walker, e *entry) bool {
	days := int64(w.now().Sub(e.info.ModTime) / (24 * time.Hour)) // remainder discarded
	return compare(x.cmp, days, x.n)
}

func (x newer) eval(_ *walker, e *entry) bool { return e.info.ModTime.After(x.t) }

// parser reads the expression (POSIX precedence: ( ), !, -a, -o).
type parser struct {
	env                *fsx.Env
	toks               []string
	hasPrint           bool // an action (-print, -exec): no implicit -print
	minDepth, maxDepth int
	batches            []*execBatch
}

func (p *parser) peek() string {
	if len(p.toks) == 0 {
		return ""
	}
	return p.toks[0]
}

func (p *parser) next() string {
	t := p.peek()
	if len(p.toks) > 0 {
		p.toks = p.toks[1:]
	}
	return t
}

func (p *parser) arg(primary string) (string, error) {
	if len(p.toks) == 0 {
		return "", fmt.Errorf("missing argument to `%s'", primary)
	}
	return p.next(), nil
}

func (p *parser) parse() (expr, error) {
	if len(p.toks) == 0 {
		return constant(true), nil
	}
	x, err := p.or()
	if err != nil {
		return nil, err
	}
	if len(p.toks) > 0 {
		return nil, fmt.Errorf("unexpected `%s'", p.peek())
	}
	return x, nil
}

func (p *parser) or() (expr, error) {
	x, err := p.and()
	for err == nil && (p.peek() == "-o" || p.peek() == "-or") {
		p.next()
		var y expr
		if y, err = p.and(); err == nil {
			x = or{x, y}
		}
	}
	return x, err
}

func (p *parser) and() (expr, error) {
	x, err := p.not()
	for err == nil {
		t := p.peek()
		if t == "" || t == ")" || t == "-o" || t == "-or" {
			break
		}
		if t == "-a" || t == "-and" {
			p.next()
		}
		var y expr
		if y, err = p.not(); err == nil {
			x = and{x, y}
		}
	}
	return x, err
}

func (p *parser) not() (expr, error) {
	if t := p.peek(); t == "!" || t == "-not" {
		p.next()
		x, err := p.not()
		return not{x}, err
	}
	return p.primary()
}

func (p *parser) primary() (expr, error) {
	t := p.next()
	switch t {
	case "":
		return nil, errors.New("expression expected")
	case "(":
		x, err := p.or()
		if err != nil {
			return nil, err
		}
		if p.next() != ")" {
			return nil, errors.New("missing `)'")
		}
		return x, nil
	case "-print":
		p.hasPrint = true
		return printer{}, nil
	case "-exec":
		var argv []string
		for {
			if len(p.toks) == 0 {
				return nil, errors.New("missing argument to `-exec'")
			}
			a := p.next()
			if a == ";" {
				break
			}
			if a == "+" && len(argv) > 0 && argv[len(argv)-1] == "{}" {
				if len(argv) == 1 {
					return nil, errors.New("-exec: no utility before {} +")
				}
				b := &execBatch{argv: argv[:len(argv)-1]}
				p.batches = append(p.batches, b)
				p.hasPrint = true
				return b, nil
			}
			argv = append(argv, a)
		}
		if len(argv) == 0 {
			return nil, errors.New("-exec: no utility")
		}
		p.hasPrint = true
		return execOne{argv}, nil
	case "-prune":
		return pruner{}, nil
	case "-true":
		return constant(true), nil
	case "-false":
		return constant(false), nil
	case "-name", "-iname":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		if _, err := path.Match(a, ""); err != nil {
			return nil, fmt.Errorf("%s: bad pattern `%s'", t, a)
		}
		return name{a, t == "-iname"}, nil
	case "-type":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		if len(a) != 1 || !strings.Contains("bcdlpfs", a) {
			return nil, fmt.Errorf("-type: unknown type `%s'", a)
		}
		return ftype(a[0]), nil
	case "-size":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		unit := int64(512)
		if i := len(a) - 1; i >= 0 {
			if u, ok := units[a[i]]; ok {
				unit, a = u, a[:i]
			}
		}
		cmp, n, err := number(a)
		if err != nil {
			return nil, fmt.Errorf("-size: %v", err)
		}
		return size{cmp, n, unit}, nil
	case "-mtime":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		cmp, n, err := number(a)
		if err != nil {
			return nil, fmt.Errorf("-mtime: %v", err)
		}
		return mtime{cmp, n}, nil
	case "-newer":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		st, err := p.env.FS.Stat(fsx.Resolve(p.env.Cwd, a))
		if err != nil {
			return nil, fmt.Errorf("%s: %s", a, prettyErr(err))
		}
		return newer{st.ModTime}, nil
	case "-maxdepth", "-mindepth":
		a, err := p.arg(t)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(a)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("%s: `%s' is not a non-negative integer", t, a)
		}
		if t == "-maxdepth" {
			p.maxDepth = n
		} else {
			p.minDepth = n
		}
		return constant(true), nil // a global option, true where it stands
	}
	return nil, fmt.Errorf("unknown predicate `%s'", t)
}

var units = map[byte]int64{'c': 1, 'w': 2, 'b': 512, 'k': 1 << 10, 'M': 1 << 20, 'G': 1 << 30}

// number reads n, +n or -n.
func number(s string) (cmp, n int64, err error) {
	switch {
	case strings.HasPrefix(s, "+"):
		cmp, s = 1, s[1:]
	case strings.HasPrefix(s, "-"):
		cmp, s = -1, s[1:]
	}
	n, err = strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 || s == "" || s[0] == '+' || s[0] == '-' {
		return 0, 0, fmt.Errorf("invalid number `%s'", s)
	}
	return cmp, n, nil
}

type walker struct {
	env      *fsx.Env
	expr     expr
	min, max int
	rc       int
	t        time.Time
}

func (w *walker) now() time.Time {
	if w.t.IsZero() {
		w.t = Now()
	}
	return w.t
}

func (w *walker) fail(shown string, err error) {
	fmt.Fprintf(w.env.Stderr, "find: %s: %s\n", shown, prettyErr(err))
	w.rc = exit.Fail
}

// walk evaluates the expression on shown (depth-first, pre-order) and
// descends into it if it is a directory.
func (w *walker) walk(shown, abs string, depth int) {
	info, err := w.env.FS.Stat(abs)
	if err != nil {
		w.fail(shown, err)
		return
	}
	e := &entry{shown: shown, info: info}
	if depth >= w.min {
		w.expr.eval(w, e)
	}
	if !info.IsDir || e.prune || (w.max >= 0 && depth >= w.max) {
		return
	}
	if depth >= MaxDepth {
		w.fail(shown, errors.New("too many levels of directories"))
		return
	}
	kids, err := w.env.FS.ReadDir(abs)
	if err != nil {
		w.fail(shown, err)
		return
	}
	prefix := shown
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for _, k := range kids {
		w.walk(prefix+k.Name, fsx.Join(abs, k.Name), depth+1)
	}
}

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrNotDir):
		return "not a directory"
	}
	return err.Error()
}
