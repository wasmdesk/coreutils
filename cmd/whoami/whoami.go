// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package whoami implements whoami: always root (see cmd/id).
package whoami

import (
	"github.com/wasmdesk/coreutils/cmd/id"
	"github.com/wasmdesk/coreutils/pkg/fsx"
)

// Run prints the effective user's name.
func Run(env *fsx.Env) int { return id.Whoami(env) }
