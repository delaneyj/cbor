package tests

import (
	"encoding/hex"
	"errors"
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

// TestDeterministicWriterOrder verifies that deterministic map encoding
// orders keys by their encoded byte representation, matching the
// prototype's behavior and RFC recommendations.
func TestDeterministicWriterOrder(t *testing.T) {
	m := map[string]any{"b": 1, "a": 2}
	b, err := cbor.AppendMapStrInterfaceDeterministic(nil, m)
	if err != nil {
		t.Fatalf("AppendMapStrIntfDeterministic error: %v", err)
	}
	// Expected encoding: a2 61 61 02 61 62 01
	want := mustHex(t, "a2616102616201")
	if !bytesEqual(b, want) {
		t.Fatalf("deterministic map mismatch: got %s want %s",
			hex.EncodeToString(b), hex.EncodeToString(want))
	}
}

// TestDuplicateKeyDetection validates that ReadMapNoDupBytes reports
// ErrDuplicateMapKey when a map contains duplicate keys.
func TestDuplicateKeyDetection(t *testing.T) {
	// {"a":1, "a":2}
	dup := mustHex(t, "a2616101616102")
	_, err := cbor.ReadMapNoDupBytes(dup)
	if !errors.Is(err, cbor.ErrDuplicateMapKey) {
		t.Fatalf("expected ErrDuplicateMapKey, got %v", err)
	}
}

// TestStrictModeLengthAndIndefinite exercises strict and deterministic
// decoding behaviors similar to the prototype:
//
//   - Non-canonical array length encodings produce ErrNonCanonicalLength.
//   - Indefinite-length arrays are forbidden in deterministic mode.
func TestStrictModeLengthAndIndefinite(t *testing.T) {
	// Non-canonical array length: 0x98 0x02 (array of length 2 encoded as uint8)
	nc := mustHex(t, "9802")
	r := cbor.NewReaderBytes(nc)
	r.SetStrictDecode(true)
	if _, err := r.ReadArrayHeader(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength, got %v", err)
	}

	// Indefinite array forbidden in deterministic mode: 0x9f 0xff (empty indefinite array)
	ind := mustHex(t, "9fff")
	r = cbor.NewReaderBytes(ind)
	r.SetDeterministicDecode(true)
	if _, _, err := r.ReadArrayStart(); !errors.Is(err, cbor.ErrIndefiniteForbidden) {
		t.Fatalf("expected ErrIndefiniteForbidden, got %v", err)
	}

	// Non-canonical map length: 0xb8 0x02 (map of length 2 encoded as uint8)
	ncMap := mustHex(t, "b802")
	r = cbor.NewReaderBytes(ncMap)
	r.SetStrictDecode(true)
	if _, err := r.ReadMapHeader(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength for map, got %v", err)
	}

	// Non-canonical bytes length: 0x59 0x00 0x01 0xff (byte string len=1 via uint16)
	ncBytes := mustHex(t, "590001ff")
	r = cbor.NewReaderBytes(ncBytes)
	r.SetStrictDecode(true)
	if _, err := r.ReadBytes(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength for bytes, got %v", err)
	}

	// Non-canonical text length: 0x79 0x00 0x01 0x61 (text len=1 via uint16)
	ncText := mustHex(t, "79000161")
	r = cbor.NewReaderBytes(ncText)
	r.SetStrictDecode(true)
	if _, err := r.ReadString(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength for text, got %v", err)
	}

	// Indefinite bytes forbidden in deterministic mode: 0x5f 0xff (empty indef bytes)
	indBytes := mustHex(t, "5fff")
	r = cbor.NewReaderBytes(indBytes)
	r.SetDeterministicDecode(true)
	if _, err := r.ReadBytes(); !errors.Is(err, cbor.ErrIndefiniteForbidden) {
		t.Fatalf("expected ErrIndefiniteForbidden for bytes, got %v", err)
	}

	// Indefinite text forbidden in deterministic mode: 0x7f 0xff (empty indef text)
	indText := mustHex(t, "7fff")
	r = cbor.NewReaderBytes(indText)
	r.SetDeterministicDecode(true)
	if _, err := r.ReadString(); !errors.Is(err, cbor.ErrIndefiniteForbidden) {
		t.Fatalf("expected ErrIndefiniteForbidden for text, got %v", err)
	}
}

// TestStrictModeIntegers exercises canonical integer encodings under
// strict mode. Certain non-minimal integer encodings should be
// rejected with ErrNonCanonicalLength, while canonical forms must
// continue to decode successfully.
func TestStrictModeIntegers(t *testing.T) {
	// Canonical 24: 0x18 0x18 must decode.
	canonPos := mustHex(t, "1818")
	r := cbor.NewReaderBytes(canonPos)
	r.SetStrictDecode(true)
	if v, err := r.ReadUint64(); err != nil || v != 24 {
		t.Fatalf("expected canonical uint64 24, got v=%d err=%v", v, err)
	}

	// Non-canonical 24 encoded via uint16: 0x19 0x00 0x18 should be rejected.
	ncPos := mustHex(t, "190018")
	r = cbor.NewReaderBytes(ncPos)
	r.SetStrictDecode(true)
	if _, err := r.ReadUint64(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength for non-canonical uint, got %v", err)
	}

	// Canonical -1: 0x20 must decode.
	canonNeg := mustHex(t, "20")
	r = cbor.NewReaderBytes(canonNeg)
	r.SetStrictDecode(true)
	if v, err := r.ReadInt64(); err != nil || v != -1 {
		t.Fatalf("expected canonical -1, got v=%d err=%v", v, err)
	}

	// Non-canonical -1 encoded via uint8 magnitude (0x38 0x00) should be rejected.
	ncNeg := mustHex(t, "3800")
	r = cbor.NewReaderBytes(ncNeg)
	r.SetStrictDecode(true)
	if _, err := r.ReadInt64(); !errors.Is(err, cbor.ErrNonCanonicalLength) {
		t.Fatalf("expected ErrNonCanonicalLength for non-canonical negative int, got %v", err)
	}
}

// TestStrictModeFloats exercises canonical float encodings under strict mode.
// Non-minimal float32/float64 encodings should be rejected with ErrNonCanonicalFloat.
func TestStrictModeFloats(t *testing.T) {
	// 1.0 encoded as float32 (non-canonical; should be float16).
	nc32 := cbor.AppendFloat32(nil, 1.0)
	r := cbor.NewReaderBytes(nc32)
	r.SetStrictDecode(true)
	if _, err := r.ReadFloat32(); !errors.Is(err, cbor.ErrNonCanonicalFloat) {
		t.Fatalf("expected ErrNonCanonicalFloat for non-canonical float32, got %v", err)
	}

	// 1.0 encoded as float64 (also non-canonical; should be float16).
	nc64 := cbor.AppendFloat64(nil, 1.0)
	r = cbor.NewReaderBytes(nc64)
	r.SetStrictDecode(true)
	if _, err := r.ReadFloat64(); !errors.Is(err, cbor.ErrNonCanonicalFloat) {
		t.Fatalf("expected ErrNonCanonicalFloat for non-canonical float64, got %v", err)
	}

	// A value not exactly representable in float32 (1/3) should be
	// canonically encoded as float64; strict mode should accept it.
	val := 1.0 / 3.0
	canon := cbor.AppendFloatCanonical(nil, val)
	r = cbor.NewReaderBytes(canon)
	r.SetStrictDecode(true)
	got, err := r.ReadFloat64()
	if err != nil {
		t.Fatalf("expected canonical float64 to decode, got err %v", err)
	}
	if got != val {
		t.Fatalf("float64 value mismatch: got %v want %v", got, val)
	}
}

// TestMaxContainerLen verifies that Reader enforces configured
// container size limits for arrays and maps.
func TestMaxContainerLen(t *testing.T) {
	// Array of length 3 with maxContainer=2 should fail.
	arr := mustHex(t, "83010203") // [1,2,3]
	r := cbor.NewReaderBytes(arr)
	r.SetMaxContainerLen(2)
	if _, err := r.ReadArrayHeader(); !errors.Is(err, cbor.ErrContainerTooLarge) {
		t.Fatalf("expected ErrContainerTooLarge for array, got %v", err)
	}

	// Map of length 3 with maxContainer=2 should fail.
	// {"a":1,"b":2,"c":3} -> a3 61 61 01 61 62 02 61 63 03
	mp := mustHex(t, "a3616101616202616303")
	r = cbor.NewReaderBytes(mp)
	r.SetMaxContainerLen(2)
	if _, err := r.ReadMapHeader(); !errors.Is(err, cbor.ErrContainerTooLarge) {
		t.Fatalf("expected ErrContainerTooLarge for map, got %v", err)
	}
}

// bytesEqual is a small helper to compare two byte slices without allocating.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
