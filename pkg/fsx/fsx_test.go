// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Clean covers every branch of the POSIX cleaner: empty, no-leading-slash,
// "." / "..", repeated slashes, and the "..-pops-empty" case.
func TestClean(t *testing.T) {
	cases := map[string]string{
		"":          "/",
		"/":         "/",
		"a":         "/a",
		"/a//b":     "/a/b",
		"/a/./b":    "/a/b",
		"/a/b/..":   "/a",
		"/a/b/../c": "/a/c",
		"/..":       "/",
		"/../..":    "/",
		"//":        "/",
	}
	for in, want := range cases {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// Join + Parent + Basename + Resolve are tiny but each has a branch worth
// asserting (root vs non-root, empty-cwd, absolute vs relative).
func TestPathHelpers(t *testing.T) {
	if got := Join("", "x"); got != "/x" {
		t.Errorf("Join empty dir = %q", got)
	}
	if got := Join("/", "x"); got != "/x" {
		t.Errorf("Join root = %q", got)
	}
	if got := Join("/a", "b"); got != "/a/b" {
		t.Errorf("Join /a + b = %q", got)
	}
	if got := Parent("/"); got != "/" {
		t.Errorf("Parent / = %q", got)
	}
	if got := Parent("/a"); got != "/" {
		t.Errorf("Parent /a = %q", got)
	}
	if got := Parent("/a/b"); got != "/a" {
		t.Errorf("Parent /a/b = %q", got)
	}
	if got := Basename("/"); got != "/" {
		t.Errorf("Basename / = %q", got)
	}
	if got := Basename("/a/b"); got != "b" {
		t.Errorf("Basename /a/b = %q", got)
	}
	if got := Resolve("/cwd", ""); got != "/cwd" {
		t.Errorf("Resolve empty = %q", got)
	}
	if got := Resolve("/cwd", "/abs"); got != "/abs" {
		t.Errorf("Resolve abs = %q", got)
	}
	if got := Resolve("/cwd", "rel"); got != "/cwd/rel" {
		t.Errorf("Resolve rel = %q", got)
	}
}

// MemFS round-trip covers every method + every error branch.
func TestMemFSRoundtrip(t *testing.T) {
	m := NewMemFS()

	// Root stat works.
	if info, err := m.Stat("/"); err != nil || !info.IsDir {
		t.Fatalf("Stat / = %+v %v", info, err)
	}

	// Stat / ReadFile on missing path.
	if _, err := m.Stat("/nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Stat /nope err = %v", err)
	}
	if _, err := m.ReadFile("/nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ReadFile /nope err = %v", err)
	}

	// Mkdir of root fails (already exists).
	if err := m.Mkdir("/"); !errors.Is(err, ErrExists) {
		t.Errorf("Mkdir / err = %v", err)
	}
	// Mkdir under missing parent.
	if err := m.Mkdir("/no/sub"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Mkdir under missing parent err = %v", err)
	}

	// Mkdir + Mkdir-exists + Mkdir under a file (NotDir).
	if err := m.Mkdir("/dir"); err != nil {
		t.Fatalf("Mkdir /dir = %v", err)
	}
	if err := m.Mkdir("/dir"); !errors.Is(err, ErrExists) {
		t.Errorf("Mkdir existing err = %v", err)
	}
	if err := m.WriteFile("/file", []byte("hi")); err != nil {
		t.Fatalf("WriteFile /file = %v", err)
	}
	if err := m.Mkdir("/file/sub"); !errors.Is(err, ErrNotDir) {
		t.Errorf("Mkdir under file err = %v", err)
	}

	// WriteFile to root + under missing parent + over directory.
	if err := m.WriteFile("/", []byte("x")); !errors.Is(err, ErrInvalid) {
		t.Errorf("WriteFile / err = %v", err)
	}
	if err := m.WriteFile("/no/sub", []byte("x")); !errors.Is(err, ErrNotFound) {
		t.Errorf("WriteFile under missing parent err = %v", err)
	}
	if err := m.WriteFile("/file/sub", []byte("x")); !errors.Is(err, ErrNotDir) {
		t.Errorf("WriteFile under file err = %v", err)
	}
	if err := m.WriteFile("/dir", []byte("x")); !errors.Is(err, ErrIsDir) {
		t.Errorf("WriteFile over dir err = %v", err)
	}

	// ReadFile of a dir = ErrIsDir.
	if _, err := m.ReadFile("/dir"); !errors.Is(err, ErrIsDir) {
		t.Errorf("ReadFile dir err = %v", err)
	}
	if b, err := m.ReadFile("/file"); err != nil || string(b) != "hi" {
		t.Errorf("ReadFile /file = %q %v", b, err)
	}

	// MkdirAll: existing dir = no-op, missing chain creates, over file = err.
	if err := m.MkdirAll("/"); err != nil {
		t.Errorf("MkdirAll / = %v", err)
	}
	if err := m.MkdirAll("/dir"); err != nil {
		t.Errorf("MkdirAll existing = %v", err)
	}
	if err := m.MkdirAll("/a/b/c"); err != nil {
		t.Fatalf("MkdirAll new chain = %v", err)
	}
	if !mustDir(t, m, "/a/b/c") {
		t.Errorf("/a/b/c not a dir")
	}
	// MkdirAll over an existing intermediate dir exercises the in-loop
	// "continue" branch (existing dir, keep walking).
	if err := m.MkdirAll("/a/b/c/d/e"); err != nil {
		t.Errorf("MkdirAll extending existing chain = %v", err)
	}
	if err := m.MkdirAll("/file"); !errors.Is(err, ErrExists) {
		t.Errorf("MkdirAll over file err = %v", err)
	}
	// MkdirAll where an intermediate path is a regular file.
	_ = m.WriteFile("/blocker", []byte(""))
	if err := m.MkdirAll("/blocker/sub"); !errors.Is(err, ErrExists) {
		t.Errorf("MkdirAll through file err = %v", err)
	}

	// ReadDir / + on missing + on a file.
	entries, err := m.ReadDir("/")
	if err != nil || len(entries) == 0 {
		t.Errorf("ReadDir / = %v %v", entries, err)
	}
	if _, err := m.ReadDir("/nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ReadDir missing err = %v", err)
	}
	if _, err := m.ReadDir("/file"); !errors.Is(err, ErrNotDir) {
		t.Errorf("ReadDir file err = %v", err)
	}
	// Listing a sub-dir with peer paths NOT under the prefix exercises the
	// HasPrefix-mismatch skip branch (e.g. ReadDir(/dir) with /file present).
	if _, err := m.ReadDir("/dir"); err != nil {
		t.Errorf("ReadDir /dir = %v", err)
	}

	// Remove of root and missing.
	if err := m.Remove("/"); !errors.Is(err, ErrInvalid) {
		t.Errorf("Remove / err = %v", err)
	}
	if err := m.Remove("/missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove missing err = %v", err)
	}
	// Remove non-empty dir.
	_ = m.MkdirAll("/p/q")
	if err := m.Remove("/p"); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("Remove non-empty err = %v", err)
	}
	if err := m.Remove("/p/q"); err != nil {
		t.Errorf("Remove empty dir = %v", err)
	}
	if err := m.Remove("/p"); err != nil {
		t.Errorf("Remove now-empty parent = %v", err)
	}

	// RemoveAll of root + missing + tree.
	if err := m.RemoveAll("/"); !errors.Is(err, ErrInvalid) {
		t.Errorf("RemoveAll / err = %v", err)
	}
	if err := m.RemoveAll("/never-existed"); err != nil {
		t.Errorf("RemoveAll missing = %v", err)
	}
	_ = m.MkdirAll("/wipe/inner")
	_ = m.WriteFile("/wipe/inner/x.txt", []byte("x"))
	if err := m.RemoveAll("/wipe"); err != nil {
		t.Errorf("RemoveAll /wipe = %v", err)
	}
	if _, err := m.Stat("/wipe"); !errors.Is(err, ErrNotFound) {
		t.Errorf("/wipe still present: %v", err)
	}
}

// mustDir helper for the round-trip assertions.
func mustDir(t *testing.T, m *MemFS, p string) bool {
	t.Helper()
	info, err := m.Stat(p)
	if err != nil {
		return false
	}
	return info.IsDir
}

// OSFS round-trip: covers every method against a tempdir + drives the error
// branches (missing path, dir-vs-file, non-empty Remove). Uses t.TempDir()
// so the test self-cleans.
func TestOSFSRoundtrip(t *testing.T) {
	dir := t.TempDir()
	osf := NewOSFS()

	// Stat the tempdir itself.
	if info, err := osf.Stat(dir); err != nil || !info.IsDir {
		t.Fatalf("Stat dir = %+v %v", info, err)
	}

	// Stat / ReadFile on missing.
	missing := filepath.Join(dir, "missing")
	if _, err := osf.Stat(missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Stat missing err = %v", err)
	}
	if _, err := osf.ReadFile(missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("ReadFile missing err = %v", err)
	}

	// Mkdir / MkdirAll / WriteFile / ReadFile / ReadDir.
	sub := filepath.Join(dir, "sub")
	if err := osf.Mkdir(sub); err != nil {
		t.Fatalf("Mkdir = %v", err)
	}
	if err := osf.Mkdir(sub); !errors.Is(err, ErrExists) {
		t.Errorf("Mkdir existing err = %v", err)
	}
	deep := filepath.Join(dir, "a", "b", "c")
	if err := osf.MkdirAll(deep); err != nil {
		t.Fatalf("MkdirAll = %v", err)
	}
	file := filepath.Join(sub, "f.txt")
	if err := osf.WriteFile(file, []byte("hi")); err != nil {
		t.Fatalf("WriteFile = %v", err)
	}
	if b, err := osf.ReadFile(file); err != nil || string(b) != "hi" {
		t.Errorf("ReadFile = %q %v", b, err)
	}
	if _, err := osf.ReadFile(sub); !errors.Is(err, ErrIsDir) {
		t.Errorf("ReadFile of dir err = %v", err)
	}
	if err := osf.WriteFile(sub, []byte("x")); !errors.Is(err, ErrIsDir) {
		t.Errorf("WriteFile over dir err = %v", err)
	}
	entries, err := osf.ReadDir(sub)
	if err != nil || len(entries) != 1 || entries[0].Name != "f.txt" {
		t.Errorf("ReadDir = %v %v", entries, err)
	}
	if _, err := osf.ReadDir(missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("ReadDir missing err = %v", err)
	}
	if _, err := osf.ReadDir(file); !errors.Is(err, ErrNotDir) {
		t.Errorf("ReadDir file err = %v", err)
	}

	// Remove: non-empty dir, then the file, then the empty dir.
	if err := osf.Remove(sub); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("Remove non-empty err = %v", err)
	}
	if err := osf.Remove(file); err != nil {
		t.Errorf("Remove file = %v", err)
	}
	if err := osf.Remove(sub); err != nil {
		t.Errorf("Remove empty dir = %v", err)
	}
	if err := osf.Remove(missing); !errors.Is(err, ErrNotFound) {
		t.Errorf("Remove missing err = %v", err)
	}

	// RemoveAll: missing is nil; tree drops cleanly.
	if err := osf.RemoveAll(missing); err != nil {
		t.Errorf("RemoveAll missing = %v", err)
	}
	tree := filepath.Join(dir, "tree")
	_ = osf.MkdirAll(filepath.Join(tree, "inner"))
	_ = osf.WriteFile(filepath.Join(tree, "inner", "x"), []byte("x"))
	if err := osf.RemoveAll(tree); err != nil {
		t.Errorf("RemoveAll tree = %v", err)
	}
	if _, err := os.Stat(tree); !os.IsNotExist(err) {
		t.Errorf("tree still present: %v", err)
	}

	// Drive the translate() pass-through branch for non-sentinel errors via a
	// ReadDir of a Stat-clean path whose contents disappear under the call.
	// We construct that by deleting the dir between Stat and ReadDir using a
	// custom flow: write a file, ask ReadDir on it (already covered above), so
	// we additionally drive translate(nil-in) to exercise the early-return.
	if got := translate(nil); got != nil {
		t.Errorf("translate(nil) = %v", got)
	}
	// errExist branch on Mkdir (translate maps fs.ErrExist).
	if err := osf.Mkdir(dir); !errors.Is(err, ErrExists) {
		t.Errorf("Mkdir on existing root err = %v", err)
	}
	// translate fallthrough: a non-os error returned unchanged.
	custom := errors.New("custom")
	if got := translate(custom); got != custom {
		t.Errorf("translate(custom) = %v, want passthrough", got)
	}

	// Drive the post-Stat ReadDir/Remove failure branches: a directory that
	// Stat sees cleanly but whose listing fails (a TOCTOU race or an EACCES on
	// POSIX). We inject that failure through the osReadDir seam so the branch is
	// reachable on every platform, including Windows, where the read-only file
	// attribute does not deny directory listing the way chmod 0 does on POSIX.
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatalf("mkdir locked = %v", err)
	}
	saved := osReadDir
	osReadDir = func(string) ([]os.DirEntry, error) { return nil, os.ErrPermission }
	defer func() { osReadDir = saved }()
	// ReadDir-after-Stat failure (Stat works, listing is denied).
	if _, err := osf.ReadDir(locked); err == nil {
		t.Errorf("ReadDir on locked dir succeeded, want err")
	}
	// Remove-after-Stat failure on a directory with the same problem.
	if err := osf.Remove(locked); err == nil {
		t.Errorf("Remove on locked dir succeeded, want err")
	}
}
