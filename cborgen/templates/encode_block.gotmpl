{{/*
Encode blocks for multi-statement field encoders, used when the simple
EncodeExpr hook is not sufficient (e.g. maps and slices that need
inline loops). These are shape-based (map/slice element types), not
per-field, mirroring msgp-style codegen.

Templates:
  encodeMapUint64PtrMarshaler - map[uint64]*T where *T has MarshalCBOR
  encodeMapUint64Uint64       - map[uint64]uint64
  encodeMapStrStr             - map[string]string
  encodeMapStrValueMarshaler  - map[string]T where T has MarshalCBOR
  encodeMapStrPtrMarshaler    - map[string]*T where *T has MarshalCBOR
  encodeMapStrScalar          - map[string]S where S is a scalar
  encodeSlicePtrMarshaler     - []*T where *T has MarshalCBOR
  encodeSliceValueMarshaler   - []T where T has MarshalCBOR
  encodeSliceScalar           - []S where S is a scalar (bool/int/float/string)

Inputs:
  .FieldRef   - "x.F" reference to the Go field
  .KeyName    - CBOR map/array key name
  .GoField    - Go field name (for variable suffixes)
  .ElemVar    - Loop variable name used for slice elements
  .AppendFunc - Append* helper name for scalar slices
*/}}

{{define "encodeMapUint64PtrMarshaler"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendUint64"}}(b, k)
		if v == nil {
			b = {{rt "AppendNil"}}(b)
		} else {
			b, err = v.MarshalCBOR(b)
			if err != nil { return b, err }
		}
	}
{{end}}

{{define "encodeMapUint64Uint64"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendUint64"}}(b, k)
		b = {{rt "AppendUint64"}}(b, v)
	}
{{end}}

{{define "encodeMapStrStr"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendString"}}(b, k)
		b = {{rt "AppendString"}}(b, v)
	}
{{end}}

{{define "encodeMapStrValueMarshaler"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendString"}}(b, k)
		b, err = v.MarshalCBOR(b)
		if err != nil { return b, err }
	}
{{end}}

{{define "encodeMapStrPtrMarshaler"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendString"}}(b, k)
		if v == nil {
			b = {{rt "AppendNil"}}(b)
		} else {
			b, err = v.MarshalCBOR(b)
			if err != nil { return b, err }
		}
	}
{{end}}

{{define "encodeMapStrScalar"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendMapHeader"}}(b, uint32(len({{.FieldRef}})))
	for k, v := range {{.FieldRef}} {
		b = {{rt "AppendString"}}(b, k)
		b = {{.AppendFunc}}(b, v)
	}
{{end}}

{{define "encodeSlicePtrMarshaler"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendArrayHeader"}}(b, uint32(len({{.FieldRef}})))
	for _, {{.ElemVar}} := range {{.FieldRef}} {
		if {{.ElemVar}} == nil {
			b = {{rt "AppendNil"}}(b)
		} else {
			b, err = {{.ElemVar}}.MarshalCBOR(b)
			if err != nil { return b, err }
		}
	}
{{end}}

{{define "encodeSliceValueMarshaler"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendArrayHeader"}}(b, uint32(len({{.FieldRef}})))
	for i := range {{.FieldRef}} {
		b, err = {{.FieldRef}}[i].MarshalCBOR(b)
		if err != nil { return b, err }
	}
{{end}}

{{define "encodeSliceScalar"}}
	b = {{rt "AppendString"}}(b, "{{.KeyName}}")
	b = {{rt "AppendArrayHeader"}}(b, uint32(len({{.FieldRef}})))
	for _, v := range {{.FieldRef}} {
		b = {{.AppendFunc}}(b, v)
	}
{{end}}
