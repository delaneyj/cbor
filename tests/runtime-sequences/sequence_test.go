package tests

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

func TestCBORSequenceBytesHelpers(t *testing.T) {
	// Build two items: string "hi" and int 42.
	it1 := cbor.AppendString(nil, "hi")
	it2 := cbor.AppendInt64(nil, 42)
	seq := cbor.AppendSequence(nil, it1, it2)

	// SplitSequenceBytes should return the two items as-is.
	items, err := cbor.SplitSequenceBytes(seq)
	if err != nil {
		t.Fatalf("SplitSequenceBytes error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	s, rem, err := cbor.ReadStringBytes(items[0])
	if err != nil || len(rem) != 0 || s != "hi" {
		t.Fatalf("first item mismatch: s=%q rem=%d err=%v", s, len(rem), err)
	}

	v, rem, err := cbor.ReadInt64Bytes(items[1])
	if err != nil || len(rem) != 0 || v != 42 {
		t.Fatalf("second item mismatch: v=%d rem=%d err=%v", v, len(rem), err)
	}

	// ForEachSequenceBytes should iterate over each item in order.
	i := 0
	err = cbor.ForEachSequenceBytes(seq, func(item []byte) error {
		switch i {
		case 0:
			s, rem, err := cbor.ReadStringBytes(item)
			if err != nil || len(rem) != 0 || s != "hi" {
				t.Fatalf("ForEachSequence first item mismatch")
			}
		case 1:
			v, rem, err := cbor.ReadInt64Bytes(item)
			if err != nil || len(rem) != 0 || v != 42 {
				t.Fatalf("ForEachSequence second item mismatch")
			}
		}
		i++
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachSequenceBytes error: %v", err)
	}
	if i != 2 {
		t.Fatalf("ForEachSequenceBytes expected 2 items, got %d", i)
	}
}

