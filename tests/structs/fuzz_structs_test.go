package structs

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// FuzzDecodeSafeTrusted exercises the generated DecodeSafe/DecodeTrusted
// entrypoints for a few representative structs to ensure they do not
// panic on arbitrary inputs.
func FuzzDecodeSafeTrusted(f *testing.F) {
	// Seed with valid encodings for known-good values.
	seedPerson := &Person{Name: "Alice", Age: 30, Data: []byte{1, 2, 3}}
	if b, err := seedPerson.MarshalCBOR(nil); err == nil {
		f.Add(b)
	}
	seedScalars := &Scalars{S: "s", B: true, I: 1}
	if b, err := seedScalars.MarshalCBOR(nil); err == nil {
		f.Add(b)
	}
	seedContainers := &Containers{}
	if b, err := seedContainers.MarshalCBOR(nil); err == nil {
		f.Add(b)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in struct fuzz: %v", r)
			}
		}()
		// Quickly screen out inputs that would require
		// excessively large container allocations by using the
		// runtime Reader with a max container length.
		r := cbor.NewReaderBytes(data)
		r.SetMaxContainerLen(1 << 16)
		if err := r.Skip(); err != nil {
			return
		}

		// Person
		var p1, p2 Person
		_, _ = p1.DecodeSafe(data)
		_, _ = p2.DecodeTrusted(data)

		// Scalars
		var s1, s2 Scalars
		_, _ = s1.DecodeSafe(data)
		_, _ = s2.DecodeTrusted(data)

		// Containers
		var c1, c2 Containers
		_, _ = c1.DecodeSafe(data)
		_, _ = c2.DecodeTrusted(data)
	})
}
