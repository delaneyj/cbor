package structs

import (
	"testing"
	"time"
)

type scalarsDecoder struct {
	name   string
	decode func(dst *Scalars, b []byte) ([]byte, error)
}

var scalarsDecoders = []scalarsDecoder{
	{
		name:   "DecodeSafe",
		decode: (*Scalars).DecodeSafe,
	},
	{
		name:   "DecodeTrusted",
		decode: (*Scalars).DecodeTrusted,
	},
}

type nestedDecoder struct {
	name   string
	decode func(dst *Nested, b []byte) ([]byte, error)
}

var nestedDecoders = []nestedDecoder{
	{
		name:   "DecodeSafe",
		decode: (*Nested).DecodeSafe,
	},
	{
		name:   "DecodeTrusted",
		decode: (*Nested).DecodeTrusted,
	},
}

func TestScalarsRoundTripSafeAndTrusted(t *testing.T) {
	orig := &Scalars{
		S:    "hello",
		B:    true,
		I:    -1,
		I8:   -8,
		I16:  -16,
		I32:  -32,
		I64:  -64,
		U:    1,
		U8:   8,
		U16:  16,
		U32:  32,
		U64:  64,
		F32:  1.5,
		F64:  2.5,
		Data: []byte{1, 2, 3, 4},
		T:    time.Unix(123456789, 0).UTC(),
		D:    5 * time.Second,
		Ints: []int{1, 2, 3},
		Names: []string{"a", "b"},
		Scores: map[string]int{"alice": 10, "bob": 20},
	}

	b, err := orig.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR error: %v", err)
	}

	for _, tc := range scalarsDecoders {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dst Scalars
			rest, err := tc.decode(&dst, b)
			if err != nil {
				t.Fatalf("%s error: %v", tc.name, err)
			}
			if len(rest) != 0 {
				t.Fatalf("%s leftover bytes: %d", tc.name, len(rest))
			}
			if dst.S != orig.S || dst.B != orig.B || dst.I != orig.I || dst.I8 != orig.I8 || dst.I16 != orig.I16 || dst.I32 != orig.I32 || dst.I64 != orig.I64 || dst.U != orig.U || dst.U8 != orig.U8 || dst.U16 != orig.U16 || dst.U32 != orig.U32 || dst.U64 != orig.U64 || dst.F32 != orig.F32 || dst.F64 != orig.F64 || string(dst.Data) != string(orig.Data) || !dst.T.Equal(orig.T) || dst.D != orig.D || !equalInts(dst.Ints, orig.Ints) || !equalStrings(dst.Names, orig.Names) || !equalIntMap(dst.Scores, orig.Scores) {
				t.Fatalf("%s mismatch: got %+v, want %+v", tc.name, dst, orig)
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) { return false }
	for i := range a { if a[i] != b[i] { return false } }
	return true
}

func equalIntMap(a, b map[string]int) bool {
	if len(a) != len(b) { return false }
	for k, v := range a {
		if b[k] != v { return false }
	}
	return true
}


func TestNestedRoundTripSafeAndTrusted(t *testing.T) {
	orig := &Nested{
		ID: "nested-1",
		Base: Scalars{
			S:    "base",
			B:    true,
			I:    10,
			I8:   -8,
			I16:  -16,
			I32:  -32,
			I64:  -64,
			U:    11,
			U8:   12,
			U16:  13,
			U32:  14,
			U64:  15,
			F32:  3.5,
			F64:  4.5,
			Data: []byte{9, 8, 7},
			T:    time.Unix(222222222, 0).UTC(),
			D:    10 * time.Second,
		},
		Ptr: &Scalars{
			S:    "ptr",
			B:    false,
			I:    -10,
			I8:   1,
			I16:  2,
			I32:  3,
			I64:  4,
			U:    21,
			U8:   22,
			U16:  23,
			U32:  24,
			U64:  25,
			F32:  5.5,
			F64:  6.5,
			Data: []byte{5, 6, 7},
			T:    time.Unix(333333333, 0).UTC(),
			D:    15 * time.Second,
		},
	}

	b, err := orig.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR error: %v", err)
	}

	for _, tc := range nestedDecoders {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dst Nested
			rest, err := tc.decode(&dst, b)
			if err != nil {
				t.Fatalf("%s error: %v", tc.name, err)
			}
			if len(rest) != 0 {
				t.Fatalf("%s leftover bytes: %d", tc.name, len(rest))
			}
			if dst.ID != orig.ID {
				t.Fatalf("%s ID mismatch: got %q, want %q", tc.name, dst.ID, orig.ID)
			}
			if !dst.Base.T.Equal(orig.Base.T) || dst.Base.D != orig.Base.D || !dst.Ptr.T.Equal(orig.Ptr.T) || dst.Ptr.D != orig.Ptr.D {
				t.Fatalf("%s time fields mismatch: got %+v, want %+v", tc.name, dst, orig)
			}
		})
	}
}
