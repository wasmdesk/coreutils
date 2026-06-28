module github.com/wasmdesk/coreutils

go 1.26.4

require github.com/go-ruby-regexp/regexp v0.0.0-20260622075631-3a277cf71b81

// go-ruby-regexp is a sibling org under bring-up. We pin to the same
// pseudo-version wasmbox already consumes, and replace to the local
// checkout so coreutils + wasmbox iterate in lockstep without a proxy
// round-trip. Drop this replace once a tagged release is cut.
replace github.com/go-ruby-regexp/regexp => ../../go-ruby-regexp/regexp
