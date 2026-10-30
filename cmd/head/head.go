// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package head

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run is POSIX head: head [-c number | -n number] [file...]. It copies the
// first number lines (default 10), or bytes with -c, of each input; with
// several, each is preceded by "==> file <==" (a blank line before all but
// the first). The obsolescent -number form means -n number. Standard
// input is read only as far as needed, so `yes | head -n 1` ends.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	n, bytesMode := int64(10), false
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			paths = append(paths, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			paths = append(paths, a)
			continue
		}
		opt, val := a[1], a[2:]
		if opt >= '0' && opt <= '9' { // -number
			opt, val = 'n', a[1:]
		}
		if opt != 'n' && opt != 'c' {
			fmt.Fprintf(env.Stderr, "head: invalid option -- '%c'\n", opt)
			return exit.Usage
		}
		if val == "" {
			i++
			if i >= len(args) {
				fmt.Fprintf(env.Stderr, "head: option requires an argument -- '%c'\n", opt)
				return exit.Usage
			}
			val = args[i]
		}
		v, err := strconv.ParseInt(val, 10, 64)
		if err != nil || v < 0 {
			fmt.Fprintf(env.Stderr, "head: invalid number: %q\n", val)
			return exit.Usage
		}
		n, bytesMode = v, opt == 'c'
	}
	if len(paths) == 0 {
		paths = []string{fsx.StdinOperand}
	}
	rc := exit.Ok
	first := true
	for _, p := range paths {
		var r io.Reader
		if p == fsx.StdinOperand {
			r = env.Stdin
			if r == nil {
				r = bytes.NewReader(nil)
			}
		} else {
			data, err := env.FS.ReadFile(fsx.Resolve(env.Cwd, p))
			if err != nil {
				fmt.Fprintf(env.Stderr, "head: %s: %s\n", p, prettyErr(err))
				rc = exit.Fail
				continue
			}
			r = bytes.NewReader(data)
		}
		if len(paths) > 1 {
			if !first {
				fmt.Fprintln(env.Stdout)
			}
			fmt.Fprintf(env.Stdout, "==> %s <==\n", p)
		}
		first = false
		if err := copyHead(env.Stdout, r, n, bytesMode); err != nil {
			fmt.Fprintf(env.Stderr, "head: %s: %v\n", p, err)
			rc = exit.Fail
		}
	}
	return rc
}

// copyHead copies the first n lines (or bytes) of r to w, reading no
// further than it needs to.
func copyHead(w io.Writer, r io.Reader, n int64, bytesMode bool) error {
	if bytesMode {
		_, err := io.Copy(w, io.LimitReader(r, n))
		return err
	}
	br := bufio.NewReader(r)
	for ; n > 0; n-- {
		line, err := br.ReadSlice('\n')
		if len(line) > 0 {
			if _, werr := w.Write(line); werr != nil {
				return werr
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			n++ // the same line goes on
			continue
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
	return nil
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
