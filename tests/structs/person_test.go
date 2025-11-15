package structs

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

type personDecoder struct {
	name   string
	decode func(dst *Person, b []byte) ([]byte, error)
}

var personDecoders = []personDecoder{
	{
		name:   "DecodeSafe",
		decode: (*Person).DecodeSafe,
	},
	{
		name:   "DecodeTrusted",
		decode: (*Person).DecodeTrusted,
	},
}

func TestPersonRoundTripSafeAndTrusted(t *testing.T) {
	orig := &Person{
		Name: "Alice",
		Age:  42,
		Data: []byte{1, 2, 3},
	}

	b, err := orig.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR error: %v", err)
	}

	for _, tc := range personDecoders {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dst Person
			rest, err := tc.decode(&dst, b)
			if err != nil {
				t.Fatalf("%s error: %v", tc.name, err)
			}
			if len(rest) != 0 {
				t.Fatalf("%s leftover bytes: %d", tc.name, len(rest))
			}
			if dst.Name != orig.Name || dst.Age != orig.Age || string(dst.Data) != string(orig.Data) {
				t.Fatalf("%s mismatch: got %+v, want %+v", tc.name, dst, orig)
			}
		})
	}
}

func TestPersonOmitEmptyAge(t *testing.T) {
	p := &Person{
		Name: "Bob",
		Age:  0,
		Data: []byte{10, 11},
	}

	b, err := p.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR error: %v", err)
	}

	sz, rest, err := cbor.ReadMapHeaderBytes(b)
	if err != nil {
		t.Fatalf("ReadMapHeaderBytes error: %v", err)
	}
	if int(sz) < 2 {
		t.Fatalf("expected at least 2 keys, got %d", sz)
	}

	foundAge := false
	for i := uint32(0); i < sz; i++ {
		key, v, err := cbor.ReadStringBytes(rest)
		if err != nil {
			t.Fatalf("ReadStringBytes key error: %v", err)
		}
		if key == "age" {
			foundAge = true
		}
		v, err = cbor.Skip(v)
		if err != nil {
			t.Fatalf("Skip value error: %v", err)
		}
		rest = v
	}

	if foundAge {
		t.Fatalf("age field should be omitted when zero")
	}

	for _, tc := range personDecoders {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var dst Person
			rest, err := tc.decode(&dst, b)
			if err != nil {
				t.Fatalf("%s error: %v", tc.name, err)
			}
			if len(rest) != 0 {
				t.Fatalf("%s leftover bytes: %d", tc.name, len(rest))
			}
			if dst.Name != p.Name || dst.Age != 0 || string(dst.Data) != string(p.Data) {
				t.Fatalf("%s mismatch: got %+v, want %+v", tc.name, dst, p)
			}
		})
	}

}
