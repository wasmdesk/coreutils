<p align="center"><img src="https://raw.githubusercontent.com/wasmdesk/brand/main/social/coreutils.png" alt="wasmdesk/coreutils" width="720"></p>

# coreutils — wasmdesk

[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Status](https://img.shields.io/badge/status-v0.3%20%C2%B7%2042%20tools-1a7f37)](#tools)

**A pure-Go (no cgo) GNU-coreutils-style command suite** that ships in two
modes: native CLI binaries (`go install ./cmd/<tool>`) and in-process
builtins for the wasmbox terminal client (the same `Run` powers both).

## Quickstart

```bash
# Native CLIs:
go install ./cmd/pwd ./cmd/echo ./cmd/cat ./cmd/ls ./cmd/mkdir \
           ./cmd/rmdir ./cmd/rm ./cmd/cp ./cmd/mv ./cmd/touch \
           ./cmd/head ./cmd/tail ./cmd/wc ./cmd/grep ./cmd/find \
           ./cmd/sort ./cmd/uniq ./cmd/cut ./cmd/tr ./cmd/paste \
           ./cmd/nl ./cmd/tac ./cmd/rev ./cmd/fold ./cmd/expand \
           ./cmd/unexpand ./cmd/printf ./cmd/basename ./cmd/dirname \
           ./cmd/date ./cmd/seq ./cmd/sleep ./cmd/true ./cmd/false \
           ./cmd/yes ./cmd/env ./cmd/expr ./cmd/md5sum ./cmd/sha1sum \
           ./cmd/sha256sum ./cmd/base64 ./cmd/base32

pwd
echo hello
ls -l
wc -l README.md
seq 1 2 9 | paste - -
printf '%s=%s\n' user david
echo hello | sha256sum
```

For browser mode, see [wasmbox](https://github.com/wasmdesk/wasmbox) — its
terminal client embeds these tools and runs them against the shared
IndexedDB-backed VFS.

## Tools

v0.3 ships 42 utilities; the package layout (`cmd/<tool>/Run(env)`) scales
to the full ~100 GNU coreutils set without redesign.

### File / filesystem (v0.2 -- 15)

| tool   | flags supported                     |
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
| `grep` | `-i`, `-v`, `-n`, `-E` (regex via go-ruby-regexp — see "Limitations") |
| `find` | `-name GLOB` (shell-style glob)     |

### Text processing (v0.3 -- 12)

| tool       | flags supported                                     |
| ---------- | --------------------------------------------------- |
| `sort`     | `-r` reverse, `-n` numeric, `-u` unique             |
| `uniq`     | `-c` count, `-d` only dups, `-u` only singletons    |
| `cut`      | `-f LIST -d DELIM`, `-c LIST`, `-b LIST` (`N`,`N-M`,`N-`,`-M`,`N,M,...`) |
| `tr`       | `SET1 [SET2]`, `-d` delete, `-s` squeeze (ranges, `\n \t \\`) |
| `paste`    | `-d DELIM` (default tab)                            |
| `nl`       | (no flags; `%6d\t%s\n` format)                      |
| `tac`      | (no flags; reverse cat)                             |
| `rev`      | (no flags; UTF-8 aware)                             |
| `fold`     | `-w N` (default 80)                                 |
| `expand`   | `-t N` (default 8) — tabs to spaces                 |
| `unexpand` | `-t N` (default 8) — leading spaces to tabs         |
| `printf`   | `%s %d %x %o %c %%`, escapes `\n \t \\`             |

### Utility (v0.3 -- 10)

| tool       | flags supported                                     |
| ---------- | --------------------------------------------------- |
| `basename` | `PATH [SUFFIX]`                                     |
| `dirname`  | (no flags; one-or-more PATHs)                       |
| `date`     | `-d RFC3339` (default: now in UTC RFC1123)          |
| `seq`      | `N` / `A B` / `A S B` (integer)                     |
| `sleep`    | `SECONDS...` (integer or float, sums multiple)      |
| `true`     | exit 0                                              |
| `false`    | exit 1                                              |
| `yes`      | `STRING`, `-n COUNT` (default 1 to avoid wedging)   |
| `env`      | (no args; prints sorted KEY=VALUE)                  |
| `expr`     | `+ - * / %`, `= != < <= > >=`, `STR : REGEX`        |

### Crypto / encoding (v0.3 -- 5)

| tool        | flags supported                              |
| ----------- | -------------------------------------------- |
| `md5sum`    | (no flags; `<hash>  <file>` / `-` for stdin) |
| `sha1sum`   | (no flags)                                   |
| `sha256sum` | (no flags)                                   |
| `base64`    | `-d` / `--decode` (whitespace-tolerant)      |
| `base32`    | `-d` / `--decode` (whitespace-tolerant)      |

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
dispatch table. The terminal looks up `multicall.Has(name)` to decide
whether to route -- so every new entry in `multicall.registry` becomes an
in-browser builtin automatically, with no terminal-side code change. See
[wasmbox](https://github.com/wasmdesk/wasmbox)
`clients/terminal/internal/scene/shell.go` for the integration.

## Limitations (v0.3)

- `touch` does not update mtime (the wasmbox VFS has no mtime field).
- `yes` defaults to a single line; pass `-n COUNT` for repeats (browser-host
  safety -- a real `yes` would spin forever).
- `cut -b` is treated as `-c` (ASCII-only). Multi-byte byte addressing lands
  with the utf8 SIMD work.
- `tr` is byte-oriented (multi-byte runes pass through verbatim, no class
  syntax like `[:alpha:]`).
- `env` is print-only -- no `KEY=VAL CMD` execution (no subprocess support).
- `expr` accepts only the binary `A OP B` form; no parentheses.
- `printf` supports `%s %d %x %o %c %%` and escapes `\n \t \\` only.
- `sleep` uses a swappable function seam (`sleep.SleepFn`); the in-browser
  builtin needs a host-aware implementation.
- `grep` defaults to **substring** match (POSIX-fixed). Pass `-E` to switch
  to the pure-Go [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp)
  engine (Onigmo subset: alternation, classes, anchors, quantifiers,
  `(?i)` etc.). Under `-E`, `-i` is wired to the engine's case-folding
  (Unicode-correct), and a malformed pattern exits 2.
- `coreutils` depends on `go-ruby-regexp` via a `replace` directive that
  points at a sibling checkout (`../../go-ruby-regexp/regexp`) for the
  bring-up era. When go-ruby-regexp cuts a tagged release, drop the
  replace and bump the require.
- `echo` has no `-e` (escape-interpret) flag.
- `ls -a` is accepted but a no-op — the wasmbox VFS has no hidden-dot
  convention.

## Build / test

```bash
task build:all   # 42 native binaries under bin/
task test        # full suite + 100% coverage gate
task run -- ls -l README.md
```
