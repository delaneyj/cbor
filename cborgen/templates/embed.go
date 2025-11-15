package templates

import "embed"

// FS exposes the codegen templates used by cborgen
// for per-struct encode/decode generation.
//
//go:embed *.gotmpl
var FS embed.FS

