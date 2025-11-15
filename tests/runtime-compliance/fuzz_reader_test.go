package tests

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// FuzzRuntimeReaderBasic fuzzes the slice-based Reader and core
// validation entrypoints to ensure they do not panic on arbitrary
// inputs under different strict/deterministic/limit settings.
func FuzzRuntimeReaderBasic(f *testing.F) {
	// Simple CBOR seeds: {"a":1}, [1,2,3], and some junk.
	f.Add([]byte{0xa1, 0x61, 0x61, 0x01})       // map {"a":1}
	f.Add([]byte{0x83, 0x01, 0x02, 0x03})       // array [1,2,3]
	f.Add([]byte{0x9f, 0x01, 0x02, 0xff})       // indef array [1,2]
	f.Add([]byte{0xff, 0x00, 0x01, 0x02, 0x03}) // invalid start

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in Reader fuzz: %v", r)
			}
		}()

		configs := []struct {
			strict bool
			det    bool
			maxLen uint32
		}{
			{false, false, 0},
			{true, false, 0},
			{false, true, 0},
			{true, true, 0},
			{true, true, 4},
		}

		for _, cfg := range configs {
			// Validate well-formedness once per config to exercise strict
			// container checks and Skip-based traversal.
			_, _ = cbor.ValidateWellFormedBytes(data)

			r := cbor.NewReaderBytes(data)
			r.SetStrictDecode(cfg.strict)
			r.SetDeterministicDecode(cfg.det)
			if cfg.maxLen > 0 {
				r.SetMaxContainerLen(cfg.maxLen)
			}

			// Exercise a few Reader methods; ignore errors.
			_, _ = r.ReadArrayHeader()
			_, _ = r.ReadMapHeader()
			_, _ = r.ReadString()
			_, _ = r.ReadBytes()
			_, _ = r.ReadInt64()
			_, _ = r.ReadUint64()
			_, _ = r.ReadFloat64()
			_ = r.Skip()
		}
	})
}

