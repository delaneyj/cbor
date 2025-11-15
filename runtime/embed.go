package cbor

import "embed"

// FS exposes the canonical CBOR runtime implementation sources used by cborgen.
//
//go:embed *.go
var FS embed.FS
