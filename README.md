# github.com/delaneyj/cbor

CBOR support for Go with a focus on **clear safety modes** and an ergonomic code generator.

> Status: this repository is being brought over from a prototype. The `cborgen` CLI is wired up, but the underlying generator implementation is still being ported.

---

**Contents**

- [What is CBOR?](#what-is-cbor)
- [Safety Modes: Safe vs Trusted](#safety-modes-safe-vs-trusted)
- [Using `cborgen` as a Go 1.24+ tool](#using-cborgen-as-a-go-124-tool)
  - [Install as a tool](#install-as-a-tool)
  - [Command-line usage](#command-line-usage)
  - [Using `cborgen` with `go generate`](#using-cborgen-with-go-generate)
  - [Behavior with files vs directories](#behavior-with-files-vs-directories)
- [Alternative: `go run` / `go install` usage](#alternative-go-run--go-install-usage)
 - [JSON ↔ CBOR interop](#json--cbor-interop)

---

## What is CBOR?

CBOR (Concise Binary Object Representation) is a binary serialization format
that plays a similar role to JSON, but is:

- **More compact** – binary encoding avoids the overhead of text
  representations.
- **Well-typed** – supports integers, floats, byte strings, text strings,
  arrays, maps, tagged values, and more.
- **Standardized** – defined by [RFC 8949], with a well-defined notion of
  *well-formed* CBOR documents.

This project aims to provide:

- A core `cbor` package for encoding/decoding Go values.
- A generator tool, `cborgen`, that produces high-performance encoders and
  decoders tailored to your types.

[RFC 8949]: https://www.rfc-editor.org/rfc/rfc8949

---

## Safety Modes: Safe vs Trusted

The key design choice in this library is that **safety and performance
trade‑offs are explicit**.

At decode time we distinguish between two modes. Importantly, these are not
runtime switches; **code generation produces separate, optimized paths for each
mode**.

- **Safe mode**
  - Validates UTF‑8 for text strings.
  - Uses **safe string conversions** (allocating new strings rather than
    reinterpreting byte slices).
  - Enforces CBOR **well-formedness** for the entire document before decoding.
  - Intended for inputs that may be attacker-controlled or otherwise untrusted.

- **Trusted mode**
  - **Skips UTF‑8 validation** for text strings.
  - Uses **zero-copy string conversions** (unsafe reinterpreting of byte slices
    as strings) where possible for maximum speed and minimal allocations.
  - May skip whole-document well-formedness checks, relying on the decoder’s
    structural checks instead.
  - Intended **only** for data that is fully trusted and immutable for the
    lifetime of the decoded strings (e.g. in-process caches, known-good blobs).

The generator (`cborgen`) emits **two decode entrypoints for each generated
type**, each with its own optimized implementation:

- `DecodeSafe` – a path compiled specifically for safe decoding.
- `DecodeTrusted` – a path compiled specifically for trusted/optimized
  decoding (including unsafe string and other fast-path optimizations).

This makes the trust boundary explicit in your application code:

```go
// From untrusted input (e.g. network, user data):
var msg MyType
_, err := msg.DecodeSafe(buf)

// From trusted, in-process caches or pre-validated blobs:
_, err := msg.DecodeTrusted(buf)
```

Unsafe optimizations (zero-copy strings, skipped validation) live **only** in
the Trusted path.

---

## Using `cborgen` as a Go 1.24+ tool

The recommended way to consume `cborgen` in Go 1.24+ is as a module-aware
tool pinned in your own module’s `go.mod`.

### Install as a tool

In your module (the project where you want to generate code), run:

```bash
go get -tool github.com/delaneyj/cbor/cborgen@latest
```

This adds a tool declaration to your `go.mod`, and lets you invoke `cborgen`
via `go tool`:

```bash
go tool cborgen -h
```

You can pin to a specific version instead of `latest`:

```bash
go get -tool github.com/delaneyj/cbor/cborgen@v0.1.0
```

### Command-line usage

Once installed as a tool, you can run:

```bash
go tool cborgen -i mytypes.go
```

Flags:

- `-i, --input`  – Go file or directory to process.
  - Default: `$GOFILE` (as set by `go generate`).
- `-o, --output` – Output file path (file mode only).
  - Default: `{input}_cbor.go`.
- `-v, --verbose` – Enable verbose diagnostics.

Examples:

```bash
# Generate for a single file; writes mytypes_cbor.go next to mytypes.go
go tool cborgen -i mytypes.go

# Explicit output path
go tool cborgen -i mytypes.go -o internal/gen/mytypes_cbor.go
```

### Using `cborgen` with `go generate`

In a Go source file in your module, add a `go generate` directive:

```go
//go:generate go tool cborgen
```

Because `cborgen` defaults `--input` to `$GOFILE`, running

```bash
go generate ./...
```

will, for each file with that directive:

- Read the current file as input.
- Generate a companion `*_cbor.go` file next to it.

If you prefer to be explicit:

```go
//go:generate go tool cborgen -i $GOFILE
```

### Behavior with files vs directories

`cborgen` behaves slightly differently depending on what `--input` points to:

- **File input** (`-i mytypes.go`)
  - Default output: `mytypes_cbor.go` in the same directory.
  - You may override the output path with `-o`.

- **Directory input** (`-i ./internal/model`)
  - For each Go source file in the directory:
    - Includes files ending in `.go`.
    - Excludes `*_test.go` and `*_cbor.go`.
  - Generates a corresponding `{file}_cbor.go` next to each included file.
  - `--output` is **not allowed** in directory mode and will result in an
    error.

Internally, the generator builds on the core `cbor` package and emits:

- High-performance encoders/decoders for your types.
- Both Safe and Trusted decode wrappers per type, wired to the options
  described above.

---

## Alternative: `go run` / `go install` usage

If you don’t want to use the `tool` mechanism, you can still use `cborgen`
directly as a module command.

**One-off runs** (no install):

```bash
go run github.com/delaneyj/cbor/cborgen@latest -i mytypes.go
```

**Installed binary**:

```bash
go install github.com/delaneyj/cbor/cborgen@latest

cborgen -i mytypes.go
```

**With `go generate`** using `go run`:

```go
//go:generate go run github.com/delaneyj/cbor/cborgen@latest
```

The semantics of `-i/-o/-v` and file vs directory inputs are the same as when
invoked via `go tool`.

---

## JSON ↔ CBOR interop

The runtime package provides helpers to convert between JSON and CBOR using a
wrapper convention compatible with the prototype and the RFC examples.

Key APIs (in `github.com/delaneyj/cbor/runtime`):

- `FromJSONBytes(js []byte) ([]byte, error)`
  - Parses a JSON document into a Go value tree (`map[string]any`, `[]any`,
    `json.Number`, etc.).
  - Recognizes wrapper objects and emits CBOR semantic tags and structures.
  - Returns encoded CBOR bytes.
- `ToJSONBytes(b []byte) (js []byte, rest []byte, err error)`
  - Decodes a single CBOR item from `b` and renders it as JSON.
  - For tagged values, produces either plain JSON (e.g. RFC3339 strings) or
    wrapper objects as described below.

Supported wrappers include (non‑exhaustive):

- Time and numeric wrappers:
  - `{ "$rfc3339": string }` → tag(0) RFC3339 time; JSON side renders as
    a plain JSON string.
  - `{ "$epoch": number }` → tag(1) epoch seconds; JSON side renders as
    RFC3339 time string.
  - `{ "$decimal": [exp, "mant"] }` → tag(4) decimal fraction.
  - `{ "$bigfloat": [exp, "mant"] }` → tag(5) bigfloat.
- Binary and URI wrappers:
  - `{ "$base64url": string }` → tag(21) byte string.
  - `{ "$base64": string }` → tag(22) byte string.
  - `{ "$base16": string }` → tag(23) byte string.
  - `{ "$cbor": string }` → tag(24) embedded CBOR.
  - `{ "$uri": string }` → tag(32) URI; JSON side renders as plain string.
- Text wrappers:
  - `{ "$base64urlstr": string }` → tag(33) text string.
  - `{ "$base64str": string }` → tag(34) text string.
  - `{ "$regex": string }` → tag(35) regex pattern.
  - `{ "$mime": string }` → tag(36) MIME message.
- UUID and self-describe:
  - `{ "$uuid": string }` → tag(37) UUID.
  - `{ "$selfdescribe": true }` → self-describe CBOR tag (55799).
- Generic tag:
  - `{ "$tag": N, "$": value }` → arbitrary tag `N` applied to the inner
    value.

Round‑tripping JSON through `FromJSONBytes` → `ToJSONBytes` (with whitespace
and map-order normalization) is tested in `tests/json-interop`. The JSON side
is stable for the wrappers above, so you can rely on these shapes when
persisting or exchanging JSON representations of tagged CBOR values.
