// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package tr

import (
	"fmt"
	"io"

	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run translates / deletes / squeezes characters in env.Stdin and writes the
// result to env.Stdout. Forms:
//
//	tr SET1 SET2   translate (each byte in SET1 -> corresponding byte in SET2)
//	tr -d SET1     delete every byte in SET1
//	tr -s SET1     squeeze adjacent duplicates of bytes in SET1
//	tr -d SET1 SET2 (illegal in GNU; we reject as Usage)
//
// Character ranges (a-z) and the escape \n / \t / \\ are honoured. v0 is
// byte-oriented; UTF-8 multi-byte characters are passed through verbatim.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	del, squeeze := false, false
	var positional []string
	for _, a := range args {
		switch a {
		case "-d":
			del = true
		case "-s":
			squeeze = true
		default:
			positional = append(positional, a)
		}
	}
	if del && len(positional) != 1 {
		fmt.Fprintln(env.Stderr, "tr: -d takes exactly one SET")
		return exit.Usage
	}
	if !del && len(positional) != 1 && len(positional) != 2 {
		fmt.Fprintln(env.Stderr, "tr: requires one or two SET operands")
		return exit.Usage
	}
	if !del && !squeeze && len(positional) != 2 {
		fmt.Fprintln(env.Stderr, "tr: translate takes exactly two SETs")
		return exit.Usage
	}
	set1 := expand(positional[0])
	var set2 []byte
	if !del && len(positional) == 2 {
		set2 = expand(positional[1])
	}
	if env.Stdin == nil {
		return exit.Ok
	}
	data, err := io.ReadAll(env.Stdin)
	if err != nil {
		fmt.Fprintf(env.Stderr, "tr: stdin: %s\n", err)
		return exit.Fail
	}
	out := transform(data, set1, set2, del, squeeze)
	env.Stdout.Write(out)
	return exit.Ok
}

// expand resolves "a-z" range syntax + a few escapes into a flat byte slice.
func expand(s string) []byte {
	var out []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				out = append(out, '\n')
				i++
				continue
			case 't':
				out = append(out, '\t')
				i++
				continue
			case '\\':
				out = append(out, '\\')
				i++
				continue
			}
		}
		if i+2 < len(s) && s[i+1] == '-' {
			lo := s[i]
			hi := s[i+2]
			if lo <= hi {
				for b := lo; b <= hi; b++ {
					out = append(out, b)
				}
				i += 2
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

// transform applies the requested transformation to data.
func transform(data, set1, set2 []byte, del, squeeze bool) []byte {
	in := make(map[byte]bool, len(set1))
	for _, b := range set1 {
		in[b] = true
	}
	if del {
		out := make([]byte, 0, len(data))
		for _, b := range data {
			if in[b] {
				continue
			}
			out = append(out, b)
		}
		if squeeze {
			out = squeezeRun(out, in)
		}
		return out
	}
	if len(set2) > 0 {
		// Build translation map. If set2 is shorter, repeat its last byte.
		xlat := make(map[byte]byte, len(set1))
		for i, b := range set1 {
			j := i
			if j >= len(set2) {
				j = len(set2) - 1
			}
			xlat[b] = set2[j]
		}
		out := make([]byte, len(data))
		for i, b := range data {
			if v, ok := xlat[b]; ok {
				out[i] = v
			} else {
				out[i] = b
			}
		}
		if squeeze {
			// In the two-SET form `-s` squeezes characters from SET2 (the
			// translated set), matching GNU tr.
			in2 := make(map[byte]bool, len(set2))
			for _, b := range set2 {
				in2[b] = true
			}
			out = squeezeRun(out, in2)
		}
		return out
	}
	// squeeze-only
	return squeezeRun(data, in)
}

// squeezeRun collapses adjacent duplicate bytes that belong to set.
func squeezeRun(data []byte, set map[byte]bool) []byte {
	if len(data) == 0 {
		return data
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[0])
	for _, b := range data[1:] {
		if b == out[len(out)-1] && set[b] {
			continue
		}
		out = append(out, b)
	}
	return out
}
