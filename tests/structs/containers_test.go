package structs

import (
	"testing"
	"time"

	cbor "github.com/delaneyj/cbor/runtime"
)

type containersDecoder struct {
	name   string
	decode func(dst *Containers, b []byte) ([]byte, error)
}

var containersDecoders = []containersDecoder{
	{
		name:   "DecodeSafe",
		decode: (*Containers).DecodeSafe,
	},
	{
		name:   "DecodeTrusted",
		decode: (*Containers).DecodeTrusted,
	},
}

func TestContainersRoundTripSafeAndTrusted(t *testing.T) {
	base := Scalars{
		S:    "base",
		B:    true,
		I:    1,
		I8:   -8,
		I16:  -16,
		I32:  -32,
		I64:  -64,
		U:    10,
		U8:   11,
		U16:  12,
		U32:  13,
		U64:  14,
		F32:  1.5,
		F64:  2.5,
		Data: []byte{1, 2, 3},
		T:    time.Unix(123456, 0).UTC(),
		D:    3 * time.Second,
	}
	ptr := Scalars{
		S:    "ptr",
		B:    false,
		I:    2,
		I8:   8,
		I16:  16,
		I32:  32,
		I64:  64,
		U:    20,
		U8:   21,
		U16:  22,
		U32:  23,
		U64:  24,
		F32:  3.5,
		F64:  4.5,
		Data: []byte{4, 5, 6},
		T:    time.Unix(654321, 0).UTC(),
		D:    7 * time.Second,
	}
	orig := &Containers{
		Items: []Scalars{base, ptr},
		Ptrs:  []*Scalars{&base, &ptr},
		Map:   map[string]Scalars{"a": base, "b": ptr},
		PtrMap: map[string]*Scalars{"x": &base, "y": &ptr},
	}

	// Sanity-check encode paths for individual fields.
	if _, err := cbor.AppendInterface(nil, orig.Items); err != nil {
		t.Fatalf("AppendIntf Items error type: %T", err)
	}
	if _, err := cbor.AppendInterface(nil, orig.Ptrs); err != nil {
		t.Fatalf("AppendIntf Ptrs error type: %T", err)
	}
	if _, err := cbor.AppendInterface(nil, orig.Map); err != nil {
		t.Fatalf("AppendIntf Map error type: %T", err)
	}
	if _, err := cbor.AppendInterface(nil, orig.PtrMap); err != nil {
		t.Fatalf("AppendIntf PtrMap error type: %T", err)
	}

	b, err := orig.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR error type: %T", err)
	}

	for _, tc := range containersDecoders {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dst Containers
			rest, err := tc.decode(&dst, b)
			if err != nil {
				t.Fatalf("%s error: %v", tc.name, err)
			}
			if len(rest) != 0 {
				t.Fatalf("%s leftover bytes: %d", tc.name, len(rest))
			}
			if len(dst.Items) != len(orig.Items) || len(dst.Ptrs) != len(orig.Ptrs) || len(dst.Map) != len(orig.Map) || len(dst.PtrMap) != len(orig.PtrMap) {
				t.Fatalf("%s container lengths mismatch: got %+v want %+v", tc.name, dst, orig)
			}
			// Spot-check a few fields to ensure struct elements were decoded.
			if dst.Items[0].S != orig.Items[0].S || dst.Items[1].I != orig.Items[1].I {
				t.Fatalf("%s Items mismatch: got %+v want %+v", tc.name, dst.Items, orig.Items)
			}
			if dst.Ptrs[0] == nil || dst.Ptrs[1] == nil || dst.Ptrs[0].S != orig.Ptrs[0].S || dst.Ptrs[1].I != orig.Ptrs[1].I {
				t.Fatalf("%s Ptrs mismatch: got %+v want %+v", tc.name, dst.Ptrs, orig.Ptrs)
			}
			if dst.Map["a"].S != orig.Map["a"].S || dst.Map["b"].I != orig.Map["b"].I {
				t.Fatalf("%s Map mismatch: got %+v want %+v", tc.name, dst.Map, orig.Map)
			}
			if dst.PtrMap["x"] == nil || dst.PtrMap["y"] == nil || dst.PtrMap["x"].S != orig.PtrMap["x"].S || dst.PtrMap["y"].I != orig.PtrMap["y"].I {
				t.Fatalf("%s PtrMap mismatch: got %+v want %+v", tc.name, dst.PtrMap, orig.PtrMap)
			}
		})
	}
}
