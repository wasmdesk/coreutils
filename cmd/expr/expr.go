// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package expr

import (
	"fmt"
	"strconv"

	onigmo "github.com/go-ruby-regexp/regexp"
	"github.com/wasmdesk/coreutils/pkg/exit"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run evaluates a small integer/string expression and prints the result.
// Supports binary forms:
//
//	A + B   A - B   A * B   A / B   A % B          (integer arithmetic)
//	A = B   A != B  A < B   A <= B  A > B   A >= B (numeric if both ints, else lex)
//	STR : REGEX                                   (anchored match length)
//
// Exit code matches GNU expr: 0 if the result is non-zero / non-empty, 1
// otherwise; Usage on a parse error.
func Run(env *fsx.Env) int {
	args := env.Args
	if len(args) > 0 {
		args = args[1:]
	}
	if len(args) != 3 {
		fmt.Fprintln(env.Stderr, "expr: usage: expr A OP B")
		return exit.Usage
	}
	a, op, b := args[0], args[1], args[2]
	switch op {
	case "+", "-", "*", "/", "%":
		av, aerr := strconv.Atoi(a)
		if aerr != nil {
			fmt.Fprintf(env.Stderr, "expr: non-integer argument: %q\n", a)
			return exit.Usage
		}
		bv, berr := strconv.Atoi(b)
		if berr != nil {
			fmt.Fprintf(env.Stderr, "expr: non-integer argument: %q\n", b)
			return exit.Usage
		}
		res, err := arith(av, bv, op)
		if err != nil {
			fmt.Fprintln(env.Stderr, "expr: "+err.Error())
			return exit.Fail
		}
		fmt.Fprintln(env.Stdout, res)
		if res == 0 {
			return exit.Fail
		}
		return exit.Ok
	case "=", "!=", "<", "<=", ">", ">=":
		out, truth := compare(a, b, op)
		fmt.Fprintln(env.Stdout, out)
		if !truth {
			return exit.Fail
		}
		return exit.Ok
	case ":":
		re, err := onigmo.Compile(b)
		if err != nil {
			fmt.Fprintf(env.Stderr, "expr: invalid regex: %q\n", b)
			return exit.Usage
		}
		// GNU expr's STR : REGEX is anchored at the start of STR and prints the
		// length of the matched prefix (0 -> no match -> Fail).
		md := re.Match(a)
		matchLen := 0
		if md != nil && md.Begin(0) == 0 {
			matchLen = md.End(0)
		}
		fmt.Fprintln(env.Stdout, matchLen)
		if matchLen == 0 {
			return exit.Fail
		}
		return exit.Ok
	}
	fmt.Fprintf(env.Stderr, "expr: unknown operator: %q\n", op)
	return exit.Usage
}

// arith returns op(a, b). Division and modulo by zero report an error.
func arith(a, b int, op string) (int, error) {
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	case "%":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a % b, nil
	}
	// Unreachable: callers only pass ops from the matching switch case.
	return 0, fmt.Errorf("unreachable: %q", op)
}

// compare returns ("1"/"0", truth). Uses numeric comparison when both
// operands parse as ints, lexicographic otherwise.
func compare(a, b, op string) (string, bool) {
	ai, aok := parseInt(a)
	bi, bok := parseInt(b)
	var truth bool
	if aok && bok {
		switch op {
		case "=":
			truth = ai == bi
		case "!=":
			truth = ai != bi
		case "<":
			truth = ai < bi
		case "<=":
			truth = ai <= bi
		case ">":
			truth = ai > bi
		case ">=":
			truth = ai >= bi
		}
	} else {
		switch op {
		case "=":
			truth = a == b
		case "!=":
			truth = a != b
		case "<":
			truth = a < b
		case "<=":
			truth = a <= b
		case ">":
			truth = a > b
		case ">=":
			truth = a >= b
		}
	}
	if truth {
		return "1", true
	}
	return "0", false
}

func parseInt(s string) (int, bool) {
	v, err := strconv.Atoi(s)
	return v, err == nil
}
