// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package wc

import (
	"bytes"
	"errors"
	"fmt"
	"unicode"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run counts newlines / whitespace-separated words / bytes for each named
// file. -l / -w / -c restrict the output columns. With no flag, all three
// are printed. With multiple files, a "total" trailer is emitted.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	var lFlag, wFlag, cFlag bool
	var paths []string
	for _, a := range args {
		switch a {
		case "-l":
			lFlag = true
		case "-w":
			wFlag = true
		case "-c":
			cFlag = true
		default:
			paths = append(paths, a)
		}
	}
	if !lFlag && !wFlag && !cFlag {
		lFlag, wFlag, cFlag = true, true, true
	}
	if len(paths) == 0 {
		fmt.Fprintln(env.Stderr, "wc: missing operand")
		return exit.Usage
	}
	rc := exit.Ok
	var totL, totW, totC int
	for _, p := range paths {
		abs := fsx.Resolve(env.Cwd, p)
		data, err := env.FS.ReadFile(abs)
		if err != nil {
			fmt.Fprintf(env.Stderr, "wc: %s: %s\n", p, prettyErr(err))
			rc = exit.Fail
			continue
		}
		l, w, c := count(data)
		totL += l
		totW += w
		totC += c
		printRow(env.Stdout, l, w, c, p, lFlag, wFlag, cFlag)
	}
	if len(paths) > 1 {
		printRow(env.Stdout, totL, totW, totC, "total", lFlag, wFlag, cFlag)
	}
	return rc
}

// count returns (lines, words, bytes) for data. A line is a terminated by
// '\n'; a word is a run of non-whitespace.
func count(data []byte) (lines, words, bytesCount int) {
	bytesCount = len(data)
	lines = bytes.Count(data, []byte{'\n'})
	inWord := false
	for _, r := range string(data) {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}
		if !inWord {
			words++
			inWord = true
		}
	}
	return
}

// printRow emits a single row matching the active flag set.
func printRow(w writer, l, ws, c int, label string, lFlag, wFlag, cFlag bool) {
	first := true
	emit := func(n int) {
		if !first {
			fmt.Fprint(w, " ")
		}
		fmt.Fprintf(w, "%d", n)
		first = false
	}
	if lFlag {
		emit(l)
	}
	if wFlag {
		emit(ws)
	}
	if cFlag {
		emit(c)
	}
	fmt.Fprintf(w, " %s\n", label)
}

type writer interface{ Write([]byte) (int, error) }

func prettyErr(err error) string {
	switch {
	case errors.Is(err, fsx.ErrNotFound):
		return "no such file or directory"
	case errors.Is(err, fsx.ErrIsDir):
		return "is a directory"
	}
	return err.Error()
}
