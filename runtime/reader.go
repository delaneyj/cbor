package cbor

import (
	"bytes"
	"encoding/binary"
	"math"
)

// Reader provides a minimal slice-based CBOR reader. It is intended
// for use by generated DecodeMsg implementations and operates on
// an in-memory buffer.
type Reader struct {
	buf           []byte
	strict        bool
	deterministic bool
	maxContainer  uint32
}

// NewReaderBytes constructs a Reader over the provided buffer.
func NewReaderBytes(b []byte) *Reader { return &Reader{buf: b} }

// SetStrictDecode controls whether the reader should enforce canonical
// length encodings for containers (arrays, maps, strings, bytes).
func (r *Reader) SetStrictDecode(strict bool) { r.strict = strict }

// SetDeterministicDecode controls whether certain non-deterministic
// features such as indefinite-length containers are forbidden.
func (r *Reader) SetDeterministicDecode(det bool) { r.deterministic = det }

// SetMaxContainerLen configures an upper bound on container lengths
// (arrays, maps, byte strings, text strings). A value of zero disables
// the limit. When exceeded, ErrContainerTooLarge is returned.
func (r *Reader) SetMaxContainerLen(max uint32) { r.maxContainer = max }

// Remaining returns the unread portion of the underlying buffer.
func (r *Reader) Remaining() []byte { return r.buf }

// ReadArrayHeader reads an array header and advances the buffer.
// When strict decoding is enabled, non-canonical length encodings
// (i.e., using a larger integer encoding than necessary) will be
// rejected with ErrNonCanonicalLength.
func (r *Reader) ReadArrayHeader() (uint32, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict {
		nonCanon, err := isNonCanonicalArrayLength(r.buf)
		if err != nil {
			return 0, err
		}
		if nonCanon {
			return 0, ErrNonCanonicalLength
		}
	}
	sz, rest, err := ReadArrayHeaderBytes(r.buf)
	if err != nil {
		return 0, err
	}
	if r.maxContainer > 0 && sz > r.maxContainer {
		return 0, ErrContainerTooLarge
	}
	r.buf = rest
	return sz, nil
}

// ReadArrayStart reads an array start and indicates whether it is
// indefinite-length. When deterministic decoding is enabled,
// indefinite-length arrays are rejected with ErrIndefiniteForbidden.
func (r *Reader) ReadArrayStart() (sz uint32, indefinite bool, err error) {
	sz, indef, rest, err := ReadArrayStartBytes(r.buf)
	if err != nil {
		return 0, false, err
	}
	if indef && r.deterministic {
		return 0, false, ErrIndefiniteForbidden
	}
	r.buf = rest
	return sz, indef, nil
}

// ReadMapHeader reads a map header and advances the buffer.
// In strict mode, non-canonical length encodings are rejected.
func (r *Reader) ReadMapHeader() (uint32, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict {
		nonCanon, err := isNonCanonicalMapLength(r.buf)
		if err != nil {
			return 0, err
		}
		if nonCanon {
			return 0, ErrNonCanonicalLength
		}
	}
	sz, rest, err := ReadMapHeaderBytes(r.buf)
	if err != nil {
		return 0, err
	}
	if r.maxContainer > 0 && sz > r.maxContainer {
		return 0, ErrContainerTooLarge
	}
	r.buf = rest
	return sz, nil
}

// ReadString reads a text string and advances the buffer.
// In strict mode, non-canonical length encodings are rejected.
// In deterministic mode, indefinite-length strings are forbidden.
func (r *Reader) ReadString() (string, error) {
	if len(r.buf) < 1 {
		return "", ErrShortBytes
	}
	if r.strict {
		nonCanon, err := isNonCanonicalTextLength(r.buf)
		if err != nil {
			return "", err
		}
		if nonCanon {
			return "", ErrNonCanonicalLength
		}
	}
	if r.deterministic && getMajorType(r.buf[0]) == majorTypeText && getAddInfo(r.buf[0]) == addInfoIndefinite {
		return "", ErrIndefiniteForbidden
	}
	s, rest, err := ReadStringBytes(r.buf)
	if err != nil {
		return "", err
	}
	r.buf = rest
	return s, nil
}

// ReadInt reads an int and advances the buffer.
// In strict mode, it enforces canonical integer encodings for both
// positive and negative values.
func (r *Reader) ReadInt() (int, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict {
		maj := getMajorType(r.buf[0])
		if maj == majorTypeUint || maj == majorTypeNegInt {
			nonCanon, err := isNonCanonicalLength(r.buf, maj)
			if err != nil {
				return 0, err
			}
			if nonCanon {
				return 0, ErrNonCanonicalLength
			}
		}
	}
	v, rest, err := ReadIntBytes(r.buf)
	if err != nil {
		return 0, err
	}
	r.buf = rest
	return v, nil
}

// ReadBytes reads a byte string and advances the buffer.
func (r *Reader) ReadBytes() ([]byte, error) {
	if len(r.buf) < 1 {
		return nil, ErrShortBytes
	}
	if r.strict {
		nonCanon, err := isNonCanonicalBytesLength(r.buf)
		if err != nil {
			return nil, err
		}
		if nonCanon {
			return nil, ErrNonCanonicalLength
		}
	}
	if r.deterministic && getMajorType(r.buf[0]) == majorTypeBytes && getAddInfo(r.buf[0]) == addInfoIndefinite {
		return nil, ErrIndefiniteForbidden
	}
	v, rest, err := ReadBytesBytes(r.buf, nil)
	if err != nil {
		return nil, err
	}
	r.buf = rest
	return v, nil
}

// Skip skips over the next CBOR item and advances the buffer.
func (r *Reader) Skip() error {
	rest, err := Skip(r.buf)
	if err != nil {
		return err
	}
	r.buf = rest
	return nil
}

// ReadBool reads a bool and advances the buffer.
func (r *Reader) ReadBool() (bool, error) {
	v, rest, err := ReadBoolBytes(r.buf)
	if err != nil {
		return false, err
	}
	r.buf = rest
	return v, nil
}

// ReadInt64 reads an int64 and advances the buffer.
// In strict mode, it enforces canonical integer encodings for both
// positive and negative values.
func (r *Reader) ReadInt64() (int64, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict {
		maj := getMajorType(r.buf[0])
		if maj == majorTypeUint || maj == majorTypeNegInt {
			nonCanon, err := isNonCanonicalLength(r.buf, maj)
			if err != nil {
				return 0, err
			}
			if nonCanon {
				return 0, ErrNonCanonicalLength
			}
		}
	}
	v, rest, err := ReadInt64Bytes(r.buf)
	if err != nil {
		return 0, err
	}
	r.buf = rest
	return v, nil
}

// ReadUint reads a uint and advances the buffer.
// In strict mode, it enforces canonical unsigned integer encodings.
func (r *Reader) ReadUint() (uint, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict && getMajorType(r.buf[0]) == majorTypeUint {
		nonCanon, err := isNonCanonicalLength(r.buf, majorTypeUint)
		if err != nil {
			return 0, err
		}
		if nonCanon {
			return 0, ErrNonCanonicalLength
		}
	}
	v, rest, err := ReadUintBytes(r.buf)
	if err != nil {
		return 0, err
	}
	r.buf = rest
	return v, nil
}

// ReadUint64 reads a uint64 and advances the buffer.
// In strict mode, it enforces canonical unsigned integer encodings.
func (r *Reader) ReadUint64() (uint64, error) {
	if len(r.buf) < 1 {
		return 0, ErrShortBytes
	}
	if r.strict && getMajorType(r.buf[0]) == majorTypeUint {
		nonCanon, err := isNonCanonicalLength(r.buf, majorTypeUint)
		if err != nil {
			return 0, err
		}
		if nonCanon {
			return 0, ErrNonCanonicalLength
		}
	}
	v, rest, err := ReadUint64Bytes(r.buf)
	if err != nil {
		return 0, err
	}
	r.buf = rest
	return v, nil
}

// ReadFloat32 reads a float32 and advances the buffer.
func (r *Reader) ReadFloat32() (float32, error) {
	orig := r.buf
	v, rest, err := ReadFloat32Bytes(r.buf)
	if err != nil {
		return 0, err
	}
	if r.strict {
		canon := AppendFloatCanonical(nil, float64(v))
		encLen := len(orig) - len(rest)
		if encLen < 0 || encLen > len(orig) {
			return 0, ErrShortBytes
		}
		if len(canon) != encLen || !bytes.Equal(orig[:encLen], canon) {
			return 0, ErrNonCanonicalFloat
		}
	}
	r.buf = rest
	return v, nil
}

// ReadFloat64 reads a float64 and advances the buffer.
func (r *Reader) ReadFloat64() (float64, error) {
	orig := r.buf
	v, rest, err := ReadFloat64Bytes(r.buf)
	if err != nil {
		return 0, err
	}
	if r.strict {
		canon := AppendFloatCanonical(nil, v)
		encLen := len(orig) - len(rest)
		if encLen < 0 || encLen > len(orig) {
			return 0, ErrShortBytes
		}
		if len(canon) != encLen || !bytes.Equal(orig[:encLen], canon) {
			return 0, ErrNonCanonicalFloat
		}
	}
	r.buf = rest
	return v, nil
}

// isNonCanonicalLength reports whether the leading header in b for the
// given major type uses a non-minimal integer encoding for its length
// according to RFC 8949 canonicalization rules.
func isNonCanonicalLength(b []byte, expectedMajor uint8) (bool, error) {
	if len(b) < 1 {
		return false, ErrShortBytes
	}
	if getMajorType(b[0]) != expectedMajor {
		return false, badPrefix(getMajorType(b[0]), expectedMajor)
	}
	add := getAddInfo(b[0])
	switch add {
	case addInfoIndefinite:
		// Canonicality applies to definite lengths; indefinite is
		// handled separately by deterministic mode.
		return false, nil
	case 0, 1, 2, 3, 4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		16, 17, 18, 19, 20, 21, 22, 23:
		// Direct additional info encodes 0..23 canonically.
		return false, nil
	case addInfoUint8:
		if len(b) < 2 {
			return false, ErrShortBytes
		}
		v := uint64(b[1])
		if v <= 23 {
			return true, nil
		}
		return false, nil
	case addInfoUint16:
		if len(b) < 3 {
			return false, ErrShortBytes
		}
		v := uint64(binary.BigEndian.Uint16(b[1:]))
		if v <= math.MaxUint8 {
			return true, nil
		}
		return false, nil
	case addInfoUint32:
		if len(b) < 5 {
			return false, ErrShortBytes
		}
		v := uint64(binary.BigEndian.Uint32(b[1:]))
		if v <= math.MaxUint16 {
			return true, nil
		}
		return false, nil
	case addInfoUint64:
		if len(b) < 9 {
			return false, ErrShortBytes
		}
		v := binary.BigEndian.Uint64(b[1:])
		if v <= math.MaxUint32 {
			return true, nil
		}
		return false, nil
	default:
		return false, &ErrUnsupportedType{}
	}
}

func isNonCanonicalArrayLength(b []byte) (bool, error) { return isNonCanonicalLength(b, majorTypeArray) }
func isNonCanonicalMapLength(b []byte) (bool, error)   { return isNonCanonicalLength(b, majorTypeMap) }
func isNonCanonicalBytesLength(b []byte) (bool, error) { return isNonCanonicalLength(b, majorTypeBytes) }
func isNonCanonicalTextLength(b []byte) (bool, error)  { return isNonCanonicalLength(b, majorTypeText) }
