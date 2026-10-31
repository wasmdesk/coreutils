// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package procx provides the fsx.Execer of a build: programs started
// through WASIX (github.com/go-wasmcontainers/shell/wasix/exec) on
// GOOS=wasip1, through os/exec natively, and none on js (the browser host
// brings its own, or the tools report that they cannot run programs).
package procx

import (
	"os"
	"strings"

	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// pathOf returns the PATH a command is looked up on: its own
// environment's when it has one (POSIX env: "If PATH is specified as a
// name=value operand to env, the value given shall be used in the search
// for utility"), else the caller's.
func pathOf(env []string) string {
	if env == nil {
		return os.Getenv("PATH")
	}
	path := ""
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v // the last one wins
		}
	}
	return path
}

// status maps an exit code to a shell status: -1 (killed by a signal, as
// os/exec reports it) becomes 128+SIGKILL.
func status(code int) int {
	if code < 0 {
		return 128 + 9
	}
	return code
}

// Default returns the Execer of this build, nil where programs cannot run.
func Default() fsx.Execer { return defaultExecer }
