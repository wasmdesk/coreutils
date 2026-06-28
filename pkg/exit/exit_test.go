// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package exit

import "testing"

// Constants are load-bearing: shells treat 0 as success and 1/2 as distinct
// failure classes. A reorder would silently break callers, so we pin them.
func TestExitCodes(t *testing.T) {
	if Ok != 0 {
		t.Errorf("Ok = %d, want 0", Ok)
	}
	if Fail != 1 {
		t.Errorf("Fail = %d, want 1", Fail)
	}
	if Usage != 2 {
		t.Errorf("Usage = %d, want 2", Usage)
	}
}
