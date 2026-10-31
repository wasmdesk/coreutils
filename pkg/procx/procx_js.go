// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

//go:build js

package procx

import "github.com/wasmdesk/coreutils/pkg/fsx"

// The browser cannot start processes; the host sets its own Env.Exec.
var defaultExecer fsx.Execer
