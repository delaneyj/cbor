{{/*
Decode case snippets for UnmarshalCBOR switch bodies.

Templates:
  decodeCaseBasic       - scalar types (string, bool, numbers)
  decodeCaseBytes       - []byte
  decodeCaseSliceBasic  - []T for basic scalar T
  decodeCaseMapStrBasic - map[string]T for basic scalar T
  decodeCaseSkip        - fallback: skip unknown/unsupported field

Inputs:
  .Field    - Go field name on receiver (exported)
  .VarType  - Go type for temporary (e.g. "int64")
  .ReadFunc - runtime ReadXxxBytes function to call
*/}}

{{define "decodeCaseBasic"}}
		var tmp {{.VarType}}
		tmp, v, err = {{.ReadFunc}}(v)
		if err != nil { return b, err }
		x.{{.Field}} = tmp
{{end}}

{{define "decodeCaseBytes"}}
		var tmp []byte
		tmp, v, err = {{rt "ReadBytesBytes"}}(v, nil)
		if err != nil { return b, err }
		x.{{.Field}} = tmp
{{end}}

{{define "decodeCaseSliceBasic"}}
		var sz uint32
		sz, v, err = {{rt "ReadArrayHeaderBytes"}}(v)
		if err != nil { return b, err }
		if cap(x.{{.Field}}) >= int(sz) {
			x.{{.Field}} = x.{{.Field}}[:sz]
		} else {
			x.{{.Field}} = make([]{{.VarType}}, sz)
		}
		if sz > 0 {
			_ = x.{{.Field}}[sz-1]
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var tmp {{.VarType}}
			tmp, v, err = {{.ReadFunc}}(v)
			if err != nil { return b, err }
			x.{{.Field}}[i{{.Field}}] = tmp
		}
{{end}}

{{define "decodeCaseMapStrBasic"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[string]{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key string
			key, v, err = {{rt "ReadStringBytes"}}(v)
			if err != nil { return b, err }
			var tmp {{.VarType}}
			tmp, v, err = {{.ReadFunc}}(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseMapUint64Ptr"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[uint64]*{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key uint64
			key, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			if len(v) == 0 { return b, {{rt "ErrShortBytes"}} }
			if v[0] == 0xf6 { // null
				var tmpBytes []byte
				tmpBytes, err = {{rt "ReadNilBytes"}}(v)
				if err != nil { return b, err }
				v = tmpBytes
				x.{{.Field}}[key] = nil
				continue
			}
			tmp := new({{.VarType}})
			v, err = tmp.UnmarshalCBOR(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseMapUint64Uint64"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[uint64]uint64, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key uint64
			key, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			var val uint64
			val, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = val
		}
{{end}}

{{define "decodeCaseSliceStruct"}}
		var sz uint32
		sz, v, err = {{rt "ReadArrayHeaderBytes"}}(v)
		if err != nil { return b, err }
		if cap(x.{{.Field}}) >= int(sz) {
			x.{{.Field}} = x.{{.Field}}[:sz]
		} else {
			x.{{.Field}} = make([]{{.VarType}}, sz)
		}
		if sz > 0 {
			_ = x.{{.Field}}[sz-1]
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var tmp {{.VarType}}
			v, err = (&tmp).UnmarshalCBOR(v)
			if err != nil { return b, err }
			x.{{.Field}}[i{{.Field}}] = tmp
		}
{{end}}

{{define "decodeCaseSliceStructTrusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadArrayHeaderBytes"}}(v)
		if err != nil { return b, err }
		if cap(x.{{.Field}}) >= int(sz) {
			x.{{.Field}} = x.{{.Field}}[:sz]
		} else {
			x.{{.Field}} = make([]{{.VarType}}, sz)
		}
		if sz > 0 {
			_ = x.{{.Field}}[sz-1]
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var tmp {{.VarType}}
			v, err = (&tmp).DecodeTrusted(v)
			if err != nil { return b, err }
			x.{{.Field}}[i{{.Field}}] = tmp
		}
{{end}}

{{define "decodeCaseSlicePtrStruct"}}
		var sz uint32
		sz, v, err = {{rt "ReadArrayHeaderBytes"}}(v)
		if err != nil { return b, err }
		if cap(x.{{.Field}}) >= int(sz) {
			x.{{.Field}} = x.{{.Field}}[:sz]
		} else {
			x.{{.Field}} = make([]*{{.VarType}}, sz)
		}
		if sz > 0 {
			_ = x.{{.Field}}[sz-1]
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			if x.{{.Field}}[i{{.Field}}] == nil { x.{{.Field}}[i{{.Field}}] = new({{.VarType}}) }
			v, err = x.{{.Field}}[i{{.Field}}].UnmarshalCBOR(v)
			if err != nil { return b, err }
		}
{{end}}

{{define "decodeCaseSlicePtrStructTrusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadArrayHeaderBytes"}}(v)
		if err != nil { return b, err }
		if cap(x.{{.Field}}) >= int(sz) {
			x.{{.Field}} = x.{{.Field}}[:sz]
		} else {
			x.{{.Field}} = make([]*{{.VarType}}, sz)
		}
		if sz > 0 {
			_ = x.{{.Field}}[sz-1]
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			if x.{{.Field}}[i{{.Field}}] == nil { x.{{.Field}}[i{{.Field}}] = new({{.VarType}}) }
			v, err = x.{{.Field}}[i{{.Field}}].DecodeTrusted(v)
			if err != nil { return b, err }
		}
{{end}}

{{define "decodeCaseMapStrStruct"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[string]{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key string
			key, v, err = {{rt "ReadStringBytes"}}(v)
			if err != nil { return b, err }
			var tmp {{.VarType}}
			v, err = (&tmp).UnmarshalCBOR(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseMapStrStructTrusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[string]{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key string
			key, v, err = {{rt "ReadStringBytes"}}(v)
			if err != nil { return b, err }
			var tmp {{.VarType}}
			v, err = (&tmp).DecodeTrusted(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseMapStrPtrStruct"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[string]*{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key string
			key, v, err = {{rt "ReadStringBytes"}}(v)
			if err != nil { return b, err }
			tmp := new({{.VarType}})
			v, err = tmp.UnmarshalCBOR(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseMapStrPtrStructTrusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[string]*{{.VarType}}, sz)
		} else if x.{{.Field}} != nil {
			clear(x.{{.Field}})
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key string
			key, v, err = {{rt "ReadStringBytes"}}(v)
			if err != nil { return b, err }
			tmp := new({{.VarType}})
			v, err = tmp.DecodeTrusted(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = tmp
		}
{{end}}

{{define "decodeCaseUnmarshalField"}}
		v, err = x.{{.Field}}.UnmarshalCBOR(v)
		if err != nil { return b, err }
{{end}}

{{define "decodeCasePtrUnmarshalField"}}
		if x.{{.Field}} == nil { x.{{.Field}} = new({{.VarType}}) }
		v, err = x.{{.Field}}.UnmarshalCBOR(v)
		if err != nil { return b, err }
{{end}}

{{define "decodeCaseTrustedField"}}
		v, err = (&x.{{.Field}}).DecodeTrusted(v)
		if err != nil { return b, err }
{{end}}

{{define "decodeCasePtrTrustedField"}}
		if x.{{.Field}} == nil { x.{{.Field}} = new({{.VarType}}) }
		v, err = x.{{.Field}}.DecodeTrusted(v)
		if err != nil { return b, err }
{{end}}

{{/*
Trusted map decoders for common numeric-key shapes.

These are selected by decodeCaseExprTrusted based on the field's
static type:

  - decodeCaseMapUint64PtrTrusted    : map[uint64]*T, value uses DecodeTrusted
  - decodeCaseMapUint64Uint64Trusted : map[uint64]uint64

They intentionally avoid per-entry map clearing to keep the Trusted
path as lean as possible for hot JetStream fields like Pending and
Redelivered.
*/}}

{{define "decodeCaseMapUint64PtrTrusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[uint64]*{{.VarType}}, sz)
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key uint64
			key, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			if len(v) == 0 { return b, {{rt "ErrShortBytes"}} }
			if v[0] == 0xf6 { // null
				var tmp []byte
				tmp, err = {{rt "ReadNilBytes"}}(v)
				if err != nil { return b, err }
				v = tmp
				x.{{.Field}}[key] = nil
				continue
			}
			val := new({{.VarType}})
			v, err = val.DecodeTrusted(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = val
		}
{{end}}

{{define "decodeCaseMapUint64Uint64Trusted"}}
		var sz uint32
		sz, v, err = {{rt "ReadMapHeaderBytes"}}(v)
		if err != nil { return b, err }
		if x.{{.Field}} == nil && sz > 0 {
			x.{{.Field}} = make(map[uint64]uint64, sz)
		}
		for i{{.Field}} := uint32(0); i{{.Field}} < sz; i{{.Field}}++ {
			var key uint64
			key, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			var val uint64
			val, v, err = {{rt "ReadUint64Bytes"}}(v)
			if err != nil { return b, err }
			x.{{.Field}}[key] = val
		}
{{end}}

{{define "decodeCaseSkip"}}
		v, err = {{rt "Skip"}}(v)
		if err != nil { return b, err }
{{end}}

{{define "decodeCaseStringTrusted"}}
		var tmpBytes []byte
		tmpBytes, v, err = {{rt "ReadStringZC"}}(v)
		if err != nil { return b, err }
		x.{{.Field}} = {{rt "UnsafeString"}}(tmpBytes)
{{end}}
