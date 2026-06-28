// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package exit holds the small set of exit codes every coreutils tool
// returns from Run. Defining them once avoids each tool inventing its own
// magic numbers and gives the dispatch layer a single source of truth when
// translating return values into shell-status semantics.
package exit

const (
	// Ok signals success. Every Run that completes its work returns this.
	Ok = 0
	// Fail signals a runtime error (missing file, permission denied, write
	// failure, etc.). Tools also emit a human-readable line to env.Stderr.
	Fail = 1
	// Usage signals the user passed bad arguments (unknown flag, missing
	// operand, etc.). Tools emit a usage hint to env.Stderr.
	Usage = 2
)
