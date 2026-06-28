<p align="center"><img src="https://raw.githubusercontent.com/wasmdesk/brand/main/social/coreutils.png" alt="wasmdesk/coreutils" width="720"></p>

# coreutils — wasmdesk

[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Status](https://img.shields.io/badge/status-v0%20%C2%B7%2015%20tools-1a7f37)](#tools)

**A pure-Go (no cgo) GNU-coreutils-style command suite** that ships in two
modes: native CLI binaries (`go install ./cmd/<tool>`) and in-process
builtins for the wasmbox terminal client (the same `Run` powers both).

## Quickstart

```bash
# Native CLIs:
go install ./cmd/pwd ./cmd/echo ./cmd/cat ./cmd/ls ./cmd/mkdir \
           ./cmd/rmdir ./cmd/rm ./cmd/cp ./cmd/mv ./cmd/touch \
           ./cmd/head ./cmd/tail ./cmd/wc ./cmd/grep ./cmd/find

pwd
echo hello
ls -l
wc -l README.md
```

For browser mode, see [wasmbox](https://github.com/wasmdesk/wasmbox) — its
terminal client embeds these tools and runs them against the shared
IndexedDB-backed VFS.

## Tools

v0 ships the 15 most-used utilities; the package layout
(`cmd/<tool>/Run(env)`) scales to the full ~100 GNU coreutils set without
redesign.

| tool   | flags supported (v0)                |
| ------ | ----------------------------------- |
| `pwd`  | (no flags)                          |
| `echo` | (no flags; v0 has no `-e`)          |
| `cat`  | `-n` (number lines)                 |
| `ls`   | `-l` (long), `-a` (accepted)        |
| `mkdir`| `-p` (create parents)               |
| `rmdir`| (no flags)                          |
| `rm`   | `-r`/`-R` (recursive), `-f` (force) |
| `cp`   | `-r`/`-R` (recursive)               |
| `mv`   | (no flags)                          |
| `touch`| (no flags; updates mtime is a no-op — see "Limitations") |
| `head` | `-n N` (default 10)                 |
| `tail` | `-n N` (default 10)                 |
| `wc`   | `-l` / `-w` / `-c`                  |
| `grep` | `-i`, `-v`, `-n` (substring match — see "Limitations") |
| `find` | `-name GLOB` (shell-style glob)     |

## Architecture

```
pkg/fsx     — FS interface + MemFS (tests) + OSFS (native)
              + Env (Args/Stdin/Stdout/Stderr/Cwd/FS)
pkg/exit    — Ok / Fail / Usage exit codes
cmd/<tool>/ — Run(env *fsx.Env) int + tests (100% statement coverage)
   main/    — thin native-CLI wrapper around Run
multicall/  — Dispatch(name, env) routes by tool name
              (busybox-style single binary + wasmbox builtin entry point)
```

Every tool's signature is `func Run(env *fsx.Env) int`. The same function
runs natively (where `env.FS` is an `OSFS` over real paths) and in the
browser (where `env.FS` is a wasmbox `sharedvfs` adapter over IndexedDB).

## Browser mode

The wasmbox terminal wires `multicall.Dispatch(name, env)` into its shell
dispatch table. The terminal's existing builtins (`cat`, `ls`, `cd`,
`mkdir`, `touch`, `rm`, `echo`, `pwd`) become thin shims; the new ones
(`cp`, `mv`, `rmdir`, `head`, `tail`, `wc`, `grep`, `find`) drop in
through the same path. See [wasmbox](https://github.com/wasmdesk/wasmbox)
`clients/terminal/internal/scene/shell.go` for the integration.

## Limitations (v0)

- `touch` does not update mtime (the wasmbox VFS has no mtime field).
- `grep` is **substring** only, not regex. The next iteration will hook the
  [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) engine behind
  a `-E` flag.
- `echo` has no `-e` (escape-interpret) flag.
- `ls -a` is accepted but a no-op — the wasmbox VFS has no hidden-dot
  convention.

## Build / test

```bash
task build:all   # 15 native binaries under bin/
task test        # full suite + 100% coverage gate
task run -- ls -l README.md
```
