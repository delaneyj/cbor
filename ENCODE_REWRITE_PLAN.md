# ENCODE_REWRITE_PLAN

This document describes a structural rewrite plan for cborgen's encode
path so that it matches the style and performance profile of
`tinylib/msgp` for struct codegen, with JetStream meta snapshot structs
used as the primary driving workload.

The current implementation has already been tuned incrementally:

- Primitive fields (ints, uints, floats, bools, strings) use direct
  `cbor.AppendXxx` helpers.
- `time.Time` and `time.Duration` use `cbor.AppendTime` and
  `cbor.AppendDuration`.
- `json.RawMessage` uses `cbor.AppendBytes` on its underlying bytes.
- `[]string` uses `cbor.AppendStringSlice`.
- `[]T` / `[]*T` where `T` is exported use `cbor.AppendSliceMarshaler`.
- Numeric-key maps are supported by runtime helpers so that the
  JetStream `ConsumerState` shape (`map[uint64]*Pending`,
  `map[uint64]uint64`) works.

However, there are still dynamic pieces:

- Map fields like `map[uint64]*Pending` and `map[uint64]uint64` are
  encoded via a generic map path (reflection inside `AppendInterface`),
  not via inline loops per field.
- Generic helpers like `AppendSliceMarshaler[T any]` and
  `AppendPtrMarshaler[T any]` use runtime interface assertions that are
  unnecessary when we know the field types at codegen time.

To match msgp encode style **exactly**, cborgen needs to generate code
that:

1. Emits **inline map loops per field** for known map shapes.
2. Emits **inline slice loops per field** for known slice shapes.
3. Uses only a minimal, inlinable runtime of primitive helpers
   (`AppendInt64`, `AppendUint64`, `AppendString`, `AppendBytes`,
   `AppendMapHeader`, `AppendArrayHeader`, etc.) without relying on any
   dynamic helper (`AppendInterface`) in generated code.

The following sections outline the structural changes required to
achieve that.

---

## 1. Template Model: From Expressions to Blocks

### Current state

`cborgen/templates/marshal.gotmpl` assumes that each field encode can be
expressed as a single expression (`EncodeExpr`) and wires it like this:

```gotmpl
b = cbor.AppendString(b, "{{.CBORName}}")
{{- if .EncodeExpr }}
b, err = {{.EncodeExpr}}
{{- else }}
b, err = cbor.AppendInterface(b, x.{{.GoName}})
{{- end }}
if err != nil { return b, err }
``

This model works well for:

- Primitive fields (we can plug in `cbor.AppendInt64(b, x.Fld), nil`).
- Helper-based encodes (e.g. `AppendStringSlice`).

It does **not** support multi-statement blocks like:

```go
b = cbor.AppendString(b, "pending")
b = cbor.AppendMapHeader(b, uint32(len(x.Pending)))
for k, v := range x.Pending {
    b = cbor.AppendUint64(b, k)
    if v == nil {
        b = cbor.AppendNil(b)
    } else {
        b, err = v.MarshalCBOR(b)
        if err != nil { return b, err }
    }
}
```

### Required change

Introduce a second per-field concept, `EncodeBlock`, alongside
`EncodeExpr` in `fieldSpec`:

```go
type fieldSpec struct {
    GoName        string
    CBORName      string
    OmitEmpty     bool
    ZeroCheck     string
    DecodeCaseSafe  string
    DecodeCaseTrust string
    EncodeExpr    string // existing single-expression encode
    EncodeBlock   string // new, multi-statement block encode
    Ignore        bool
}
```

Extend the marshal template to use `EncodeBlock` when present:

```gotmpl
{{- range .Fields }}
{{- if .OmitEmpty }}
    if !({{.ZeroCheck}}) {
{{- if .EncodeBlock }}
        {{.EncodeBlock}}
{{- else }}
        b = cbor.AppendString(b, "{{.CBORName}}")
        {{- if .EncodeExpr }}
        b, err = {{.EncodeExpr}}
        {{- else }}
        b, err = cbor.AppendInterface(b, x.{{.GoName}})
        {{- end }}
        if err != nil { return b, err }
{{- end }}
    }
{{- else }}
{{- if .EncodeBlock }}
    {{.EncodeBlock}}
{{- else }}
    b = cbor.AppendString(b, "{{.CBORName}}")
    {{- if .EncodeExpr }}
    b, err = {{.EncodeExpr}}
    {{- else }}
    b, err = cbor.AppendInterface(b, x.{{.GoName}})
    {{- end }}
    if err != nil { return b, err }
{{- end }}
{{- end }}
{{- end }}
```

With this in place, map fields (and any other complex patterns) can use
`EncodeBlock` to emit loops and multi-line logic, while all other fields
continue to use `EncodeExpr`.

---

## 2. Generator Changes: Map-Aware Encode Blocks

With `EncodeBlock` available, `generateStructCode` in `cborgen/core/run.go`
should be extended to recognize specific map shapes and populate
`EncodeBlock` accordingly.

For the JetStream meta snapshot structs, the important map fields are:

```go
// tests/jetstreammeta/types.go

// ConsumerState mirrors NATS ConsumerState.
type ConsumerState struct {
    Delivered   SequencePair         `json:"delivered" msg:"delivered"`
    AckFloor    SequencePair         `json:"ack_floor" msg:"ack_floor"`
    Pending     map[uint64]*Pending  `json:"pending,omitempty" msg:"pending,omitempty"`
    Redelivered map[uint64]uint64    `json:"redelivered,omitempty" msg:"redelivered,omitempty"`
}

// Stream/Consumer config snapshots use:
//   Metadata map[string]string
```

### 2.1. `map[uint64]*Pending` (ConsumerState.Pending)

Target encode block (msgp-style, but with CBOR helpers):

```go
// For field Pending in ConsumerState
b = cbor.AppendString(b, "pending")
// Map header
b = cbor.AppendMapHeader(b, uint32(len(x.Pending)))
for k, v := range x.Pending {
    b = cbor.AppendUint64(b, k)
    if v == nil {
        b = cbor.AppendNil(b)
    } else {
        b, err = v.MarshalCBOR(b)
        if err != nil { return b, err }
    }
}
```

### 2.2. `map[uint64]uint64` (ConsumerState.Redelivered)

Target encode block:

```go
b = cbor.AppendString(b, "redelivered")
b = cbor.AppendMapHeader(b, uint32(len(x.Redelivered)))
for k, v := range x.Redelivered {
    b = cbor.AppendUint64(b, k)
    b = cbor.AppendUint64(b, v)
}
```

### 2.3. `map[string]string` (Metadata)

This is already supported via `AppendMapStrStr`, but for full msgp-style
symmetry, a per-field block can be emitted:

```go
b = cbor.AppendString(b, "metadata")
b = cbor.AppendMapHeader(b, uint32(len(x.Metadata)))
for k, v := range x.Metadata {
    b = cbor.AppendString(b, k)
    b = cbor.AppendString(b, v)
}
```

### 2.4. Implementing detection in `generateStructCode`

In `generateStructCode`, where `fieldSizeExpr`, `encodeExprForField`,
and `decodeCaseExpr...` are currently invoked, add a new helper:

```go
fs.EncodeBlock = encodeBlockForField(ss.Name, fs.GoName, field.Type)
```

`encodeBlockForField` would:

- Inspect `typ` via `ast.Expr`.
- If it is a map type (`*ast.MapType`):
  - If key is `*ast.Ident` named `"uint64"` and value is `*ast.StarExpr`
    pointing at `*ast.Ident` named `"Pending"`, generate the
    `Pending` block above.
  - If key is `"uint64"` and value is `"uint64"`, generate the
    `Redelivered` block.
  - If key is `"string"` and value is `"string"`, generate `Metadata` block.
- Otherwise, return `""` to indicate no special block.

All string concatenation for code generation should be careful about
indentation and line breaks; using a `bytes.Buffer` and `fmt.Fprintf`
with explicit `\n` is recommended.

With these blocks in place, map fields will:

- Not call `AppendInterface`.
- Not use reflection (`reflect.MapKeys`).
- Not rely on generic runtime map handling.
- Match msgp’s encode style: map header + loop over entries.

---

## 3. Generator Changes: Slice Encode Blocks (Optional)

Slices are already partially specialized via `EncodeExpr` and helpers:

- `[]string` → `AppendStringSlice`.
- `[]T` / `[]*T` → `AppendSliceMarshaler[T]`.

To match msgp even more closely and eliminate the remaining interface
conversions in `AppendSliceMarshaler`, we can emit inline loops for
selected slice fields.

For example, for `[]*WriteableConsumerAssignment`:

```go
b = cbor.AppendString(b, "consumers")
b = cbor.AppendArrayHeader(b, uint32(len(x.Consumers)))
for _, ca := range x.Consumers {
    if ca == nil {
        b = cbor.AppendNil(b)
    } else {
        b, err = ca.MarshalCBOR(b)
        if err != nil { return b, err }
    }
}
```

This again requires `EncodeBlock` for slices. The detection logic in
`encodeBlockForField` could:

- Recognize `[]*SomeExportedType` where `*SomeExportedType` has a
  generated `MarshalCBOR` method.
- Generate the above block instead of using `AppendSliceMarshaler`.

Generic helpers (`AppendSliceMarshaler`, `AppendPtrMarshaler`) can then
be used only as fallbacks for less performance-critical code paths.

---

## 4. Runtime Simplification After Rewrite

Once encode blocks are in place for maps and slices in JetStream and
similar structs, many of the dynamic runtime helpers become unnecessary
for codegen-heavy workloads:

- `AppendInterface` can be retained solely for dynamic APIs
  (`AppendInterface` itself, JSON→CBOR interop), but **must not** be
  referenced in generated `MarshalCBOR` for target structs.
- `AppendSliceMarshaler` / `AppendPtrMarshaler` can be used only where
  the generator intentionally chooses not to inline, or can be removed
  entirely once all known-hot structs use inline loops.
- `AppendMapUint64Marshaler` and `AppendMapUint64Uint64` can be
  deprecated in favor of inline field-specific blocks.

The net result should be:

- For JetStream meta snapshot structs and similar workloads, the
  **encode path contains no dynamic dispatch, no reflection, and no
  use of `AppendInterface`**.
- The runtime is a set of primitive helpers (`AppendXxx`), not a dynamic
  encoder.

---

## 5. Testing and Benchmarking Plan

### 5.1. Unit tests

- Extend `tests/jetstreammeta` to:
  - Validate `MarshalCBOR` + `DecodeTrusted` roundtrip for
    `MetaSnapshot` with various fixture sizes.
  - Add property tests for map fields (`Pending`, `Redelivered`,
    `Metadata`) to ensure all entries are preserved and types roundtrip
    correctly.

### 5.2. Encode structure sanity

- Add a simple AST-based or `rg`-based check to ensure that
  `tests/jetstreammeta/types_cbor.go` **does not** contain
  `AppendInterface` calls in `MarshalCBOR` for:
  - `MetaSnapshot`
  - `WriteableStreamAssignment`
  - `WriteableConsumerAssignment`
  - `ConsumerState`

### 5.3. Benchmarks

- Use `cmd/jetstream-meta-bench` to compare:
  - CBOR (cborgen) encode for `MetaSnapshot`.
  - JSON (`encoding/json`) encode for `MetaSnapshot`.
  - MSGP (`MarshalMsg` on MsgpMetaSnapshot) encode.

Target metrics after the rewrite:

- `Allocs/op` for CBOR encode at or very close to 0 for the JetStream
  snapshot structs (same as msgp).
- `ns/op` within a small factor of msgp’s `MarshalMsg` on the same
  logical structure.

---

## 6. Summary

To **match msgp encode style** for the JetStream structs:

- The primary changes are in cborgen’s generator and templates, not in
  the runtime.
- We need to:
  - Move from per-field expressions to per-field blocks where needed.
  - Generate inline loops for `map[uint64]*Pending`, `map[uint64]uint64`,
    and `map[string]string` fields.
  - Optionally inline loops for slices of Marshaler structs.
- After this rewrite, generated `MarshalCBOR` for the JetStream meta
  structs will:
  - Contain only map/array headers, field names, primitive appends, and
    `MarshalCBOR` calls.
  - Use no reflection or dynamic `AppendInterface` calls in encode.
  - Mirror msgp’s encode structure closely, enabling a fair, apples-to-
    apples comparison between CBOR and MessagePack struct codegen.

