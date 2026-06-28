// Copyright (c) 2026 The wasmdesk/coreutils authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package fsx

import "strings"

// MemFS is the trivial pure-Go FS used by every tool's tests + by callers
// that want a sandboxed filesystem without touching disk. Keyed by Cleaned
// absolute paths.
type MemFS struct {
	nodes map[string]*memNode
}

type memNode struct {
	isDir bool
	data  []byte
}

// NewMemFS returns a fresh MemFS whose only node is the root directory.
func NewMemFS() *MemFS {
	m := &MemFS{nodes: make(map[string]*memNode)}
	m.nodes["/"] = &memNode{isDir: true}
	return m
}

// Stat implements FS.
func (m *MemFS) Stat(p string) (FileInfo, error) {
	c := Clean(p)
	n, ok := m.nodes[c]
	if !ok {
		return FileInfo{}, ErrNotFound
	}
	return FileInfo{Name: Basename(c), Size: int64(len(n.data)), IsDir: n.isDir}, nil
}

// ReadFile implements FS.
func (m *MemFS) ReadFile(p string) ([]byte, error) {
	c := Clean(p)
	n, ok := m.nodes[c]
	if !ok {
		return nil, ErrNotFound
	}
	if n.isDir {
		return nil, ErrIsDir
	}
	out := make([]byte, len(n.data))
	copy(out, n.data)
	return out, nil
}

// WriteFile implements FS.
func (m *MemFS) WriteFile(p string, data []byte) error {
	c := Clean(p)
	if c == "/" {
		return ErrInvalid
	}
	parent := Parent(c)
	pn, ok := m.nodes[parent]
	if !ok {
		return ErrNotFound
	}
	if !pn.isDir {
		return ErrNotDir
	}
	if existing, ok := m.nodes[c]; ok && existing.isDir {
		return ErrIsDir
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	m.nodes[c] = &memNode{isDir: false, data: cp}
	return nil
}

// Mkdir implements FS.
func (m *MemFS) Mkdir(p string) error {
	c := Clean(p)
	if c == "/" {
		return ErrExists
	}
	if _, exists := m.nodes[c]; exists {
		return ErrExists
	}
	parent := Parent(c)
	pn, ok := m.nodes[parent]
	if !ok {
		return ErrNotFound
	}
	if !pn.isDir {
		return ErrNotDir
	}
	m.nodes[c] = &memNode{isDir: true}
	return nil
}

// MkdirAll implements FS.
func (m *MemFS) MkdirAll(p string) error {
	c := Clean(p)
	if c == "/" {
		return nil
	}
	if n, ok := m.nodes[c]; ok {
		if n.isDir {
			return nil
		}
		return ErrExists
	}
	// Walk from the root, creating missing dirs along the way.
	parts := strings.Split(strings.TrimPrefix(c, "/"), "/")
	cur := ""
	for _, seg := range parts {
		cur += "/" + seg
		if n, ok := m.nodes[cur]; ok {
			if !n.isDir {
				return ErrExists
			}
			continue
		}
		m.nodes[cur] = &memNode{isDir: true}
	}
	return nil
}

// ReadDir implements FS.
func (m *MemFS) ReadDir(dir string) ([]FileInfo, error) {
	d := Clean(dir)
	n, ok := m.nodes[d]
	if !ok {
		return nil, ErrNotFound
	}
	if !n.isDir {
		return nil, ErrNotDir
	}
	prefix := d
	if prefix != "/" {
		prefix += "/"
	}
	out := []FileInfo{}
	for path, node := range m.nodes {
		if path == d {
			continue
		}
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := path[len(prefix):]
		if strings.Contains(rest, "/") {
			continue
		}
		out = append(out, FileInfo{Name: rest, Size: int64(len(node.data)), IsDir: node.isDir})
	}
	sortByName(out)
	return out, nil
}

// Remove implements FS.
func (m *MemFS) Remove(p string) error {
	c := Clean(p)
	if c == "/" {
		return ErrInvalid
	}
	n, ok := m.nodes[c]
	if !ok {
		return ErrNotFound
	}
	if n.isDir {
		prefix := c + "/"
		for k := range m.nodes {
			if strings.HasPrefix(k, prefix) {
				return ErrNotEmpty
			}
		}
	}
	delete(m.nodes, c)
	return nil
}

// RemoveAll implements FS.
func (m *MemFS) RemoveAll(p string) error {
	c := Clean(p)
	if c == "/" {
		return ErrInvalid
	}
	if _, ok := m.nodes[c]; !ok {
		return nil
	}
	prefix := c + "/"
	delete(m.nodes, c)
	for k := range m.nodes {
		if strings.HasPrefix(k, prefix) {
			delete(m.nodes, k)
		}
	}
	return nil
}
