package tests

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// FuzzCBORSequences fuzzes the sequence helpers to ensure they do not
// panic on arbitrary input.
func FuzzCBORSequences(f *testing.F) {
	// Seed with a simple two-item sequence.
	it1 := cbor.AppendString(nil, "hi")
	it2 := cbor.AppendInt64(nil, 42)
	seq := cbor.AppendSequence(nil, it1, it2)
	f.Add(seq)

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in sequence fuzz: %v", r)
			}
		}()

		// SplitSequenceBytes should either return an error or items; ignore error.
		items, _ := cbor.SplitSequenceBytes(data)
		_ = items

		// ForEachSequenceBytes should walk items without panicking.
		_ = cbor.ForEachSequenceBytes(data, func(item []byte) error {
			// Lightly touch the item: just try to Skip it.
			rest, _ := cbor.Skip(item)
			_ = rest
			return nil
		})
	})
}

