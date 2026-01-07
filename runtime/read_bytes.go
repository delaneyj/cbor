package cbor

import (
	"encoding/binary"
	"errors"
	"math"
	bigmath "math/big"
	"regexp"
	"time"
)

var be = binary.BigEndian

// readUintCore reads an unsigned integer with the given expected major type
func readUintCore(b []byte, expectedMajor uint8) (uint64, []byte, error) {
	if len(b) < 1 {
		return 0, b, ErrShortBytes
	}

	major := getMajorType(b[0])
	if major != expectedMajor {
		return 0, b, badPrefix(major, expectedMajor)
	}

	addInfo := getAddInfo(b[0])

	switch {
	case addInfo <= addInfoDirect:
		return uint64(addInfo), b[1:], nil
	case addInfo == addInfoUint8:
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return uint64(b[1]), b[2:], nil
	case addInfo == addInfoUint16:
		if len(b) < 3 {
			return 0, b, ErrShortBytes
		}
		return uint64(be.Uint16(b[1:])), b[3:], nil
	case addInfo == addInfoUint32:
		if len(b) < 5 {
			return 0, b, ErrShortBytes
		}
		return uint64(be.Uint32(b[1:])), b[5:], nil
	case addInfo == addInfoUint64:
		if len(b) < 9 {
			return 0, b, ErrShortBytes
		}
		return be.Uint64(b[1:]), b[9:], nil
	default:
		return 0, b, &ErrUnsupportedType{}
	}
}

// ReadMapHeaderBytes reads a map header
func ReadMapHeaderBytes(b []byte) (sz uint32, o []byte, err error) {
	if len(b) < 1 {
		return 0, b, ErrShortBytes
	}

	lead := b[0]

	// Ultra-fast paths: major type 5 (map): 0xa0-0xbb
	if lead >= 0xa0 && lead <= 0xb7 { // size 0-23
		return uint32(lead - 0xa0), b[1:], nil
	}
	if lead == 0xb8 { // size in uint8
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return uint32(b[1]), b[2:], nil
	}
	if lead == 0xb9 { // size in uint16
		if len(b) < 3 {
			return 0, b, ErrShortBytes
		}
		return uint32(be.Uint16(b[1:])), b[3:], nil
	}
	if lead == 0xba { // size in uint32
		if len(b) < 5 {
			return 0, b, ErrShortBytes
		}
		return be.Uint32(b[1:]), b[5:], nil
	}
	if lead == 0xbb { // size in uint64
		if len(b) < 9 {
			return 0, b, ErrShortBytes
		}
		u := be.Uint64(b[1:])
		if u > math.MaxUint32 {
			return 0, b, UintOverflow{Value: u, FailedBitsize: 32}
		}
		return uint32(u), b[9:], nil
	}

	major := getMajorType(lead)
	return 0, b, badPrefix(major, majorTypeMap)
}

// ReadArrayHeaderBytes reads an array header
func ReadArrayHeaderBytes(b []byte) (sz uint32, o []byte, err error) {
	if len(b) < 1 {
		return 0, b, ErrShortBytes
	}

	lead := b[0]

	// Ultra-fast paths: major type 4 (array): 0x80-0x9b
	if lead >= 0x80 && lead <= 0x97 { // size 0-23
		return uint32(lead - 0x80), b[1:], nil
	}
	if lead == 0x98 { // size in uint8
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return uint32(b[1]), b[2:], nil
	}
	if lead == 0x99 { // size in uint16
		if len(b) < 3 {
			return 0, b, ErrShortBytes
		}
		return uint32(be.Uint16(b[1:])), b[3:], nil
	}
	if lead == 0x9a { // size in uint32
		if len(b) < 5 {
			return 0, b, ErrShortBytes
		}
		return be.Uint32(b[1:]), b[5:], nil
	}
	if lead == 0x9b { // size in uint64
		if len(b) < 9 {
			return 0, b, ErrShortBytes
		}
		u := be.Uint64(b[1:])
		if u > math.MaxUint32 {
			return 0, b, UintOverflow{Value: u, FailedBitsize: 32}
		}
		return uint32(u), b[9:], nil
	}

	major := getMajorType(lead)
	return 0, b, badPrefix(major, majorTypeArray)
}

// ReadMapStartBytes reads a map start and indicates whether it is indefinite-length.
// If indefinite is true, sz is zero and rest points after the header byte (0xbf).
func ReadMapStartBytes(b []byte) (sz uint32, indefinite bool, rest []byte, err error) {
	if len(b) < 1 {
		return 0, false, b, ErrShortBytes
	}
	if b[0] == makeByte(majorTypeMap, addInfoIndefinite) {
		return 0, true, b[1:], nil
	}
	s, o, e := ReadMapHeaderBytes(b)
	return s, false, o, e
}

// ReadArrayStartBytes reads an array start and indicates whether it is indefinite-length.
// If indefinite is true, sz is zero and rest points after the header byte (0x9f).
func ReadArrayStartBytes(b []byte) (sz uint32, indefinite bool, rest []byte, err error) {
	if len(b) < 1 {
		return 0, false, b, ErrShortBytes
	}
	if b[0] == makeByte(majorTypeArray, addInfoIndefinite) {
		return 0, true, b[1:], nil
	}
	s, o, e := ReadArrayHeaderBytes(b)
	return s, false, o, e
}

// ReadBreakBytes checks whether the next byte is a break (0xff) and consumes it if so.
func ReadBreakBytes(b []byte) (rest []byte, ok bool, err error) {
	if len(b) < 1 {
		return b, false, ErrShortBytes
	}
	if b[0] == makeByte(majorTypeSimple, simpleBreak) {
		return b[1:], true, nil
	}
	return b, false, nil
}

// ReadNilBytes reads a nil value
func ReadNilBytes(b []byte) ([]byte, error) {
	if len(b) < 1 {
		return b, ErrShortBytes
	}
	if b[0] != makeByte(majorTypeSimple, simpleNull) {
		return b, ErrNotNil
	}
	return b[1:], nil
}

// ReadFloat64Bytes reads a float64
func ReadFloat64Bytes(b []byte) (f float64, o []byte, err error) {
	if len(b) < 9 {
		return 0, b, ErrShortBytes
	}
	// Ultra-fast path: direct byte comparison (0xfb = float64)
	if b[0] != 0xfb {
		return 0, b, badPrefix(getMajorType(b[0]), majorTypeSimple)
	}
	f = math.Float64frombits(be.Uint64(b[1:]))
	return f, b[9:], nil
}

// ReadFloat32Bytes reads a float32
func ReadFloat32Bytes(b []byte) (f float32, o []byte, err error) {
	if len(b) < 5 {
		return 0, b, ErrShortBytes
	}
	// Ultra-fast path: direct byte comparison (0xfa = float32)
	if b[0] != 0xfa {
		return 0, b, badPrefix(getMajorType(b[0]), majorTypeSimple)
	}
	f = math.Float32frombits(be.Uint32(b[1:]))
	return f, b[5:], nil
}

// ReadFloat16Bytes reads a float16 (IEEE 754 binary16) and returns float32
func ReadFloat16Bytes(b []byte) (f float32, o []byte, err error) {
	if len(b) < 3 {
		return 0, b, ErrShortBytes
	}
	if b[0] != 0xF9 {
		return 0, b, badPrefix(getMajorType(b[0]), majorTypeSimple)
	}
	h := binary.BigEndian.Uint16(b[1:])
	f = float16BitsToFloat32(h)
	return f, b[3:], nil
}

// ReadBoolBytes reads a bool
func ReadBoolBytes(b []byte) (bool, []byte, error) {
	if len(b) < 1 {
		return false, b, ErrShortBytes
	}
	// Ultra-fast path: direct byte comparison
	if b[0] == 0xf5 { // true
		return true, b[1:], nil
	}
	if b[0] == 0xf4 { // false
		return false, b[1:], nil
	}
	return false, b, TypeError{Method: BoolType, Encoded: getType(b[0])}
}

// ReadInt64Bytes reads an int64
func ReadInt64Bytes(b []byte) (i int64, o []byte, err error) {
	if len(b) < 1 {
		return 0, b, ErrShortBytes
	}

	lead := b[0]

	// Ultra-fast paths: direct byte pattern matching, no bit ops
	// Major type 0 (positive): 0x00-0x1b
	if lead <= 0x17 { // 0-23 direct encoding
		return int64(lead), b[1:], nil
	}
	if lead == 0x18 { // uint8 follows
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return int64(b[1]), b[2:], nil
	}
	if lead == 0x19 { // uint16 follows
		if len(b) < 3 {
			return 0, b, ErrShortBytes
		}
		return int64(be.Uint16(b[1:])), b[3:], nil
	}
	if lead == 0x1a { // uint32 follows
		if len(b) < 5 {
			return 0, b, ErrShortBytes
		}
		u := uint64(be.Uint32(b[1:]))
		return int64(u), b[5:], nil
	}
	if lead == 0x1b { // uint64 follows
		if len(b) < 9 {
			return 0, b, ErrShortBytes
		}
		u := be.Uint64(b[1:])
		if u > math.MaxInt64 {
			return 0, b, IntOverflow{Value: int64(u), FailedBitsize: 64}
		}
		return int64(u), b[9:], nil
	}

	// Major type 1 (negative): 0x20-0x3b
	if lead >= 0x20 && lead <= 0x37 { // -1 to -24 direct encoding
		return -1 - int64(lead-0x20), b[1:], nil
	}
	if lead == 0x38 { // negative, uint8 follows
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return -1 - int64(b[1]), b[2:], nil
	}
	if lead == 0x39 { // negative, uint16 follows
		if len(b) < 3 {
			return 0, b, ErrShortBytes
		}
		return -1 - int64(be.Uint16(b[1:])), b[3:], nil
	}
	if lead == 0x3a { // negative, uint32 follows
		if len(b) < 5 {
			return 0, b, ErrShortBytes
		}
		return -1 - int64(be.Uint32(b[1:])), b[5:], nil
	}
	if lead == 0x3b { // negative, uint64 follows
		if len(b) < 9 {
			return 0, b, ErrShortBytes
		}
		u := be.Uint64(b[1:])
		if u > math.MaxInt64 {
			return 0, b, IntOverflow{Value: -1, FailedBitsize: 64}
		}
		return -1 - int64(u), b[9:], nil
	}

	// Invalid major type for integer
	major := (lead >> 5) & 0x07
	return 0, b, badPrefix(major, majorTypeUint)
}

// ReadInt32Bytes reads an int32
func ReadInt32Bytes(b []byte) (i int32, o []byte, err error) {
	i64, o, err := ReadInt64Bytes(b)
	if err != nil {
		return 0, b, err
	}
	if i64 > math.MaxInt32 || i64 < math.MinInt32 {
		return 0, b, IntOverflow{Value: i64, FailedBitsize: 32}
	}
	return int32(i64), o, nil
}

// ReadInt16Bytes reads an int16
func ReadInt16Bytes(b []byte) (i int16, o []byte, err error) {
	i64, o, err := ReadInt64Bytes(b)
	if err != nil {
		return 0, b, err
	}
	if i64 > math.MaxInt16 || i64 < math.MinInt16 {
		return 0, b, IntOverflow{Value: i64, FailedBitsize: 16}
	}
	return int16(i64), o, nil
}

// ReadInt8Bytes reads an int8
func ReadInt8Bytes(b []byte) (i int8, o []byte, err error) {
	i64, o, err := ReadInt64Bytes(b)
	if err != nil {
		return 0, b, err
	}
	if i64 > math.MaxInt8 || i64 < math.MinInt8 {
		return 0, b, IntOverflow{Value: i64, FailedBitsize: 8}
	}
	return int8(i64), o, nil
}

// ReadIntBytes reads an int
func ReadIntBytes(b []byte) (i int, o []byte, err error) {
	i64, o, err := ReadInt64Bytes(b)
	if err != nil {
		return 0, b, err
	}
	return int(i64), o, nil
}

// ReadUint64Bytes reads a uint64
func ReadUint64Bytes(b []byte) (u uint64, o []byte, err error) {
	return readUintCore(b, majorTypeUint)
}

// ReadUint32Bytes reads a uint32
func ReadUint32Bytes(b []byte) (u uint32, o []byte, err error) {
	u64, o, err := readUintCore(b, majorTypeUint)
	if err != nil {
		return 0, b, err
	}
	if u64 > math.MaxUint32 {
		return 0, b, UintOverflow{Value: u64, FailedBitsize: 32}
	}
	return uint32(u64), o, nil
}

// ReadUint16Bytes reads a uint16
func ReadUint16Bytes(b []byte) (u uint16, o []byte, err error) {
	u64, o, err := readUintCore(b, majorTypeUint)
	if err != nil {
		return 0, b, err
	}
	if u64 > math.MaxUint16 {
		return 0, b, UintOverflow{Value: u64, FailedBitsize: 16}
	}
	return uint16(u64), o, nil
}

// ReadUint8Bytes reads a uint8
func ReadUint8Bytes(b []byte) (u uint8, o []byte, err error) {
	u64, o, err := readUintCore(b, majorTypeUint)
	if err != nil {
		return 0, b, err
	}
	if u64 > math.MaxUint8 {
		return 0, b, UintOverflow{Value: u64, FailedBitsize: 8}
	}
	return uint8(u64), o, nil
}

// ReadUintBytes reads a uint
func ReadUintBytes(b []byte) (u uint, o []byte, err error) {
	u64, o, err := readUintCore(b, majorTypeUint)
	if err != nil {
		return 0, b, err
	}
	return uint(u64), o, nil
}

// ReadBytesBytes reads a byte string
func ReadBytesBytes(b []byte, scratch []byte) (v []byte, o []byte, err error) {
	if len(b) < 1 {
		return nil, b, ErrShortBytes
	}
	// Indefinite form: 0x5f
	if b[0] == makeByte(majorTypeBytes, addInfoIndefinite) {
		out := scratch[:0]
		p := b[1:]
		for {
			if len(p) < 1 {
				return nil, b, ErrShortBytes
			}
			// Break?
			if p[0] == makeByte(majorTypeSimple, simpleBreak) {
				return out, p[1:], nil
			}
			// Next must be a definite-length byte string
			sz, q, e := readUintCore(p, majorTypeBytes)
			if e != nil {
				return nil, b, e
			}
			if uint64(len(q)) < sz {
				return nil, b, ErrShortBytes
			}
			out = append(out, q[:sz]...)
			p = q[sz:]
		}
	}
	lead := b[0]
	if lead >= 0x40 && lead <= 0x57 { // byte string length 0-23
		sz := int(lead & 0x1f)
		if len(b) < 1+sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], b[1:], nil
		}
		return b[1 : 1+sz], b[1+sz:], nil
	}
	switch lead {
	case 0x58: // uint8
		if len(b) < 2 {
			return nil, b, ErrShortBytes
		}
		sz := int(b[1])
		if len(b) < 2+sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], b[2:], nil
		}
		return b[2 : 2+sz], b[2+sz:], nil
	case 0x59: // uint16
		if len(b) < 3 {
			return nil, b, ErrShortBytes
		}
		sz := int(be.Uint16(b[1:]))
		if len(b) < 3+sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], b[3:], nil
		}
		return b[3 : 3+sz], b[3+sz:], nil
	case 0x5a: // uint32
		if len(b) < 5 {
			return nil, b, ErrShortBytes
		}
		sz := int(be.Uint32(b[1:]))
		if len(b) < 5+sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], b[5:], nil
		}
		return b[5 : 5+sz], b[5+sz:], nil
	case 0x5b: // uint64
		if len(b) < 9 {
			return nil, b, ErrShortBytes
		}
		u64 := be.Uint64(b[1:])
		if u64 > math.MaxInt {
			return nil, b, UintOverflow{Value: u64, FailedBitsize: 64}
		}
		sz := int(u64)
		if len(b) < 9+sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], b[9:], nil
		}
		return b[9 : 9+sz], b[9+sz:], nil
	default:
		sz, o, err := readUintCore(b, majorTypeBytes)
		if err != nil {
			return nil, b, err
		}
		if uint64(len(o)) < sz {
			return nil, b, ErrShortBytes
		}
		if sz == 0 {
			return scratch[:0], o, nil
		}
		return o[:sz], o[sz:], nil
	}
}

// ReadStringZC reads a text string zero-copy (returns slice into original buffer)
func ReadStringZC(b []byte) (v []byte, o []byte, err error) {
	if len(b) < 1 {
		return nil, b, ErrShortBytes
	}

	lead := b[0]

	// Ultra-fast path for length 0-23
	if lead >= 0x60 && lead <= 0x77 {
		sz := int(lead & 0x1f)
		if len(b) < 1+sz {
			return nil, b, ErrShortBytes
		}
		return b[1 : 1+sz], b[1+sz:], nil
	}

	// Longer strings
	var sz int
	var start int

	switch lead {
	case 0x78: // uint8
		if len(b) < 2 {
			return nil, b, ErrShortBytes
		}
		sz = int(b[1])
		start = 2
	case 0x79: // uint16
		if len(b) < 3 {
			return nil, b, ErrShortBytes
		}
		sz = int(be.Uint16(b[1:]))
		start = 3
	case 0x7a: // uint32
		if len(b) < 5 {
			return nil, b, ErrShortBytes
		}
		sz = int(be.Uint32(b[1:]))
		start = 5
	case 0x7b: // uint64
		if len(b) < 9 {
			return nil, b, ErrShortBytes
		}
		u64 := be.Uint64(b[1:])
		if u64 > math.MaxInt {
			return nil, b, UintOverflow{Value: u64, FailedBitsize: 64}
		}
		sz = int(u64)
		start = 9
	default:
		// Invalid major type
		major := getMajorType(lead)
		return nil, b, badPrefix(major, majorTypeText)
	}

	// Guard against integer overflow and out-of-bounds slicing.
	// Use subtraction form to avoid start+sz overflow when sz is near MaxInt.
	if start < 0 || start > len(b) {
		return nil, b, ErrShortBytes
	}
	if sz < 0 || sz > len(b)-start {
		return nil, b, ErrShortBytes
	}
	end := start + sz
	return b[start:end], b[end:], nil
}

// ReadStringBytes reads a text string
func ReadStringBytes(b []byte) (s string, o []byte, err error) {
	if len(b) < 1 {
		return "", b, ErrShortBytes
	}
	// Indefinite-length text string (0x7f)
	if b[0] == makeByte(majorTypeText, addInfoIndefinite) {
		p := b[1:]
		var out []byte
		for {
			if len(p) < 1 {
				return "", b, ErrShortBytes
			}
			if p[0] == makeByte(majorTypeSimple, simpleBreak) {
				if ValidateUTF8OnDecode && !isUTF8Valid(out) {
					return "", b, ErrInvalidUTF8
				}
				return string(out), p[1:], nil
			}
			chunk, q, e := ReadStringZC(p)
			if e != nil {
				return "", b, e
			}
			out = append(out, chunk...)
			p = q
		}
	}
	v, o, err := ReadStringZC(b)
	if err != nil {
		return "", b, err
	}
	if ValidateUTF8OnDecode && !isUTF8Valid(v) {
		return "", b, ErrInvalidUTF8
	}
	if UnsafeStringDecode {
		return UnsafeString(v), o, nil
	}
	return string(v), o, nil
}

// ReadMapKeyZC reads a map key expecting a text string and returns its bytes zero-copy.
// It is a thin wrapper around ReadStringZC for generated code compatibility.
func ReadMapKeyZC(b []byte) (v []byte, o []byte, err error) {
	// For CBOR, map keys are typically text. Support text zero-copy here.
	if len(b) < 1 {
		return nil, b, ErrShortBytes
	}
	if getMajorType(b[0]) != majorTypeText {
		// Fallback: treat as text anyway to surface a type error consistently
		return nil, b, TypeError{Method: StrType, Encoded: getType(b[0])}
	}
	return ReadStringZC(b)
}

// ReadSimpleValue reads a simple value and returns its numeric value.
// Returns values 0..23 (including false/true/null/undefined) directly,
// or 32..255 following a 0xf8 prefix. Float encodings are not handled here.
func ReadSimpleValue(b []byte) (val uint8, o []byte, err error) {
	if len(b) < 1 {
		return 0, b, ErrShortBytes
	}
	major := getMajorType(b[0])
	if major != majorTypeSimple {
		return 0, b, badPrefix(major, majorTypeSimple)
	}
	addInfo := getAddInfo(b[0])
	switch addInfo {
	case simpleFloat16, simpleFloat32, simpleFloat64:
		return 0, b, &ErrUnsupportedType{}
	case addInfoUint8: // 0xf8 XX
		if len(b) < 2 {
			return 0, b, ErrShortBytes
		}
		return b[1], b[2:], nil
	default:
		if addInfo <= addInfoDirect {
			return addInfo, b[1:], nil
		}
		return 0, b, &ErrUnsupportedType{}
	}
}

// ReadTimeBytes reads a time.Time (CBOR tag 1 with Unix timestamp)
func ReadTimeBytes(b []byte) (t time.Time, o []byte, err error) {
	if len(b) < 2 {
		return time.Time{}, b, ErrShortBytes
	}
	if getMajorType(b[0]) != majorTypeTag {
		return time.Time{}, b, badPrefix(getMajorType(b[0]), majorTypeTag)
	}
	tag, o, err := readUintCore(b, majorTypeTag)
	if err != nil {
		return time.Time{}, b, err
	}
	if tag != tagEpochDateTime {
		return time.Time{}, b, errors.New("cbor: expected epoch datetime tag")
	}
	if len(o) < 1 {
		return time.Time{}, b, ErrShortBytes
	}
	switch getMajorType(o[0]) {
	case majorTypeUint, majorTypeNegInt:
		sec, o2, e := ReadInt64Bytes(o)
		if e != nil {
			return time.Time{}, b, e
		}
		return time.Unix(sec, 0), o2, nil
	case majorTypeSimple:
		add := getAddInfo(o[0])
		switch add {
		case simpleFloat64:
			f, o2, e := ReadFloat64Bytes(o)
			if e != nil {
				return time.Time{}, b, e
			}
			sec := math.Floor(f)
			ns := int64(math.Round((f - sec) * 1e9))
			secs := int64(sec)
			if ns >= 1e9 {
				secs++
				ns -= 1e9
			}
			return time.Unix(secs, ns), o2, nil
		case simpleFloat32:
			f, o2, e := ReadFloat32Bytes(o)
			if e != nil {
				return time.Time{}, b, e
			}
			sec := math.Floor(float64(f))
			ns := int64(math.Round((float64(f) - sec) * 1e9))
			secs := int64(sec)
			if ns >= 1e9 {
				secs++
				ns -= 1e9
			}
			return time.Unix(secs, ns), o2, nil
		case simpleFloat16:
			f, o2, e := ReadFloat16Bytes(o)
			if e != nil {
				return time.Time{}, b, e
			}
			sec := math.Floor(float64(f))
			ns := int64(math.Round((float64(f) - sec) * 1e9))
			secs := int64(sec)
			if ns >= 1e9 {
				secs++
				ns -= 1e9
			}
			return time.Unix(secs, ns), o2, nil
		default:
			return time.Time{}, b, &ErrUnsupportedType{}
		}
	default:
		return time.Time{}, b, &ErrUnsupportedType{}
	}
}

// ReadTagBytes reads a semantic tag value (major type 6)
func ReadTagBytes(b []byte) (tag uint64, o []byte, err error) {
	tag, o, err = readUintCore(b, majorTypeTag)
	if err != nil {
		return 0, b, err
	}
	return tag, o, nil
}

// ReadRFC3339TimeBytes reads a tag(0) RFC3339 time string into time.Time
func ReadRFC3339TimeBytes(b []byte) (t time.Time, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return time.Time{}, b, err
	}
	if tag != tagDateTimeString {
		return time.Time{}, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	s, o2, err := ReadStringBytes(o)
	if err != nil {
		return time.Time{}, b, err
	}
	tt, perr := time.Parse(time.RFC3339Nano, s)
	if perr != nil {
		return time.Time{}, b, perr
	}
	return tt, o2, nil
}

// ReadBase64URLStringBytes reads tag(33) base64url text string
func ReadBase64URLStringBytes(b []byte) (s string, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return "", b, err
	}
	if tag != tagBase64URLString {
		return "", b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadStringBytes(o)
}

// ReadBase64StringBytes reads tag(34) base64 text string
func ReadBase64StringBytes(b []byte) (s string, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return "", b, err
	}
	if tag != tagBase64String {
		return "", b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadStringBytes(o)
}

// ReadURIStringBytes reads a tag(32) URI text string
func ReadURIStringBytes(b []byte) (uri string, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return "", b, err
	}
	if tag != tagURI {
		return "", b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadStringBytes(o)
}

// ReadEmbeddedCBORBytes reads tag(24) with embedded CBOR payload
func ReadEmbeddedCBORBytes(b []byte) (payload []byte, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return nil, b, err
	}
	if tag != tagCBOR {
		return nil, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadBytesBytes(o, nil)
}

// ReadBase64URLBytes reads tag(21) byte string
func ReadBase64URLBytes(b []byte) (bs []byte, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return nil, b, err
	}
	if tag != tagBase64URL {
		return nil, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadBytesBytes(o, nil)
}

// ReadBase64Bytes reads tag(22) byte string
func ReadBase64Bytes(b []byte) (bs []byte, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return nil, b, err
	}
	if tag != tagBase64 {
		return nil, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadBytesBytes(o, nil)
}

// ReadBase16Bytes reads tag(23) byte string
func ReadBase16Bytes(b []byte) (bs []byte, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return nil, b, err
	}
	if tag != tagBase16 {
		return nil, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadBytesBytes(o, nil)
}

// ReadUUIDBytes reads tag(37) UUID as 16-byte array
func ReadUUIDBytes(b []byte) (uuid [16]byte, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return uuid, b, err
	}
	if tag != 37 {
		return uuid, b, badPrefix(majorTypeTag, majorTypeTag)
	}
	bs, o2, err := ReadBytesBytes(o, nil)
	if err != nil {
		return uuid, b, err
	}
	if len(bs) != 16 {
		return uuid, b, errors.New("cbor: uuid must be 16 bytes")
	}
	copy(uuid[:], bs)
	return uuid, o2, nil
}

// ReadRegexpStringBytes reads tag(35) regular expression pattern as text string
func ReadRegexpStringBytes(b []byte) (pattern string, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return "", b, err
	}
	if tag != tagRegexp {
		return "", b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadStringBytes(o)
}

// ReadMIMEStringBytes reads tag(36) MIME message as text string
func ReadMIMEStringBytes(b []byte) (mime string, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return "", b, err
	}
	if tag != tagMIME {
		return "", b, badPrefix(majorTypeTag, majorTypeTag)
	}
	return ReadStringBytes(o)
}

// StripSelfDescribeCBOR checks for and consumes a self-describe CBOR tag (0xd9d9f7)
func StripSelfDescribeCBOR(b []byte) (rest []byte, found bool, err error) {
	if len(b) < 1 {
		return b, false, ErrShortBytes
	}
	if getMajorType(b[0]) != majorTypeTag {
		return b, false, nil
	}
	tag, o, e := ReadTagBytes(b)
	if e != nil {
		return b, false, e
	}
	if tag != tagSelfDescribeCBOR {
		return b, false, nil
	}
	return o, true, nil
}

// ReadRegexpBytes reads tag(35) and compiles the contained pattern into *regexp.Regexp
func ReadRegexpBytes(b []byte) (re *regexp.Regexp, o []byte, err error) {
	s, o, err := ReadRegexpStringBytes(b)
	if err != nil {
		return nil, b, err
	}
	r, e := regexp.Compile(s)
	if e != nil {
		return nil, b, e
	}
	return r, o, nil
}

// ReadBigIntBytes reads a bignum (tag 2 or 3) into a big.Int
func ReadBigIntBytes(b []byte) (z *bigmath.Int, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return nil, b, err
	}
	bs, o2, err := ReadBytesBytes(o, nil)
	if err != nil {
		return nil, b, err
	}
	mag := new(bigmath.Int).SetBytes(bs)
	switch tag {
	case tagPosBignum:
		return mag, o2, nil
	case tagNegBignum:
		mag.Add(mag, bigmath.NewInt(1))
		mag.Neg(mag)
		return mag, o2, nil
	default:
		return nil, b, badPrefix(majorTypeTag, majorTypeTag)
	}
}

// readCBORIntegerAsBigInt reads a CBOR integer (major type 0/1) or bignum (tags 2/3) into big.Int
func readCBORIntegerAsBigInt(b []byte) (*bigmath.Int, []byte, error) {
	if len(b) < 1 {
		return nil, b, ErrShortBytes
	}
	major := getMajorType(b[0])
	switch major {
	case majorTypeUint:
		u, o, err := readUintCore(b, majorTypeUint)
		if err != nil {
			return nil, b, err
		}
		zz := new(bigmath.Int).SetUint64(u)
		return zz, o, nil
	case majorTypeNegInt:
		i, o, err := ReadInt64Bytes(b)
		if err != nil {
			return nil, b, err
		}
		zz := bigmath.NewInt(i)
		return zz, o, nil
	case majorTypeTag:
		tag, o, err := ReadTagBytes(b)
		if err != nil {
			return nil, b, err
		}
		if tag != tagPosBignum && tag != tagNegBignum {
			return nil, b, &ErrUnsupportedType{}
		}
		bs, o2, err := ReadBytesBytes(o, nil)
		if err != nil {
			return nil, b, err
		}
		mag := new(bigmath.Int).SetBytes(bs)
		if tag == tagNegBignum {
			mag.Add(mag, bigmath.NewInt(1))
			mag.Neg(mag)
		}
		return mag, o2, nil
	default:
		return nil, b, &ErrUnsupportedType{}
	}
}

// ReadDecimalFractionBytes reads tag(4) decimal fraction [exponent, mantissa]
func ReadDecimalFractionBytes(b []byte) (exp int64, mant *bigmath.Int, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return 0, nil, b, err
	}
	if tag != tagDecimalFrac {
		return 0, nil, b, &ErrUnsupportedType{}
	}
	// Handle definite and indefinite arrays
	if len(o) < 1 {
		return 0, nil, b, ErrShortBytes
	}
	if o[0] == makeByte(majorTypeArray, addInfoIndefinite) {
		// skip header
		p := o[1:]
		// exponent
		exp, p, err = ReadInt64Bytes(p)
		if err != nil {
			return 0, nil, b, err
		}
		// mantissa
		mant, p, err = readCBORIntegerAsBigInt(p)
		if err != nil {
			return 0, nil, b, err
		}
		// expect break
		if len(p) < 1 || p[0] != makeByte(majorTypeSimple, simpleBreak) {
			return 0, nil, b, &ErrUnsupportedType{}
		}
		return exp, mant, p[1:], nil
	}
	// definite
	sz, p, err := ReadArrayHeaderBytes(o)
	if err != nil {
		return 0, nil, b, err
	}
	if sz != 2 {
		return 0, nil, b, ArrayError{Wanted: 2, Got: sz}
	}
	exp, p, err = ReadInt64Bytes(p)
	if err != nil {
		return 0, nil, b, err
	}
	mant, p, err = readCBORIntegerAsBigInt(p)
	if err != nil {
		return 0, nil, b, err
	}
	return exp, mant, p, nil
}

// ReadBigfloatBytes reads tag(5) bigfloat [exponent, mantissa]
func ReadBigfloatBytes(b []byte) (exp int64, mant *bigmath.Int, o []byte, err error) {
	tag, o, err := ReadTagBytes(b)
	if err != nil {
		return 0, nil, b, err
	}
	if tag != tagBigfloat {
		return 0, nil, b, &ErrUnsupportedType{}
	}
	if len(o) < 1 {
		return 0, nil, b, ErrShortBytes
	}
	if o[0] == makeByte(majorTypeArray, addInfoIndefinite) {
		p := o[1:]
		exp, p, err = ReadInt64Bytes(p)
		if err != nil {
			return 0, nil, b, err
		}
		mant, p, err = readCBORIntegerAsBigInt(p)
		if err != nil {
			return 0, nil, b, err
		}
		if len(p) < 1 || p[0] != makeByte(majorTypeSimple, simpleBreak) {
			return 0, nil, b, &ErrUnsupportedType{}
		}
		return exp, mant, p[1:], nil
	}
	sz, p, err := ReadArrayHeaderBytes(o)
	if err != nil {
		return 0, nil, b, err
	}
	if sz != 2 {
		return 0, nil, b, ArrayError{Wanted: 2, Got: sz}
	}
	exp, p, err = ReadInt64Bytes(p)
	if err != nil {
		return 0, nil, b, err
	}
	mant, p, err = readCBORIntegerAsBigInt(p)
	if err != nil {
		return 0, nil, b, err
	}
	return exp, mant, p, nil
}

// ReadMapNoDupBytes validates that the next CBOR item is a map and that it has no duplicate keys.
// Keys are compared by raw CBOR byte representation. Returns the bytes after the map or an error.
func ReadMapNoDupBytes(b []byte) (o []byte, err error) {
	if len(b) < 1 {
		return b, ErrShortBytes
	}
	if getMajorType(b[0]) != majorTypeMap {
		return b, badPrefix(majorTypeMap, getMajorType(b[0]))
	}
	// Indefinite-length map
	if getAddInfo(b[0]) == addInfoIndefinite {
		seen := make(map[string]struct{})
		p := b[1:]
		for {
			if len(p) < 1 {
				return b, ErrShortBytes
			}
			if p[0] == makeByte(majorTypeSimple, simpleBreak) {
				return p[1:], nil
			}
			// Capture raw key bytes
			r, err := Skip(p)
			if err != nil {
				return b, err
			}
			keyLen := len(p) - len(r)
			rawKey := p[:keyLen]
			keyStr := string(rawKey)
			if _, ok := seen[keyStr]; ok {
				return b, ErrDuplicateMapKey
			}
			seen[keyStr] = struct{}{}
			// Skip value
			r2, err := Skip(r)
			if err != nil {
				return b, err
			}
			p = r2
		}
	}
	// Definite-length map
	sz, p, err := ReadMapHeaderBytes(b)
	if err != nil {
		return b, err
	}
	seen := make(map[string]struct{}, sz)
	for i := uint32(0); i < sz; i++ {
		r, err := Skip(p)
		if err != nil {
			return b, err
		}
		keyLen := len(p) - len(r)
		rawKey := p[:keyLen]
		keyStr := string(rawKey)
		if _, ok := seen[keyStr]; ok {
			return b, ErrDuplicateMapKey
		}
		seen[keyStr] = struct{}{}
		// skip value
		p2, err := Skip(r)
		if err != nil {
			return b, err
		}
		p = p2
	}
	return p, nil
}

// ForEachSequenceBytes calls onItem for each CBOR item in a CBOR sequence buffer b.
// The item passed to onItem is a slice referencing b containing exactly one item.
func ForEachSequenceBytes(b []byte, onItem func(item []byte) error) error {
	p := b
	for len(p) > 0 {
		r, err := Skip(p)
		if err != nil {
			return err
		}
		seg := p[:len(p)-len(r)]
		if err := onItem(seg); err != nil {
			return err
		}
		p = r
	}
	return nil
}

// SplitSequenceBytes splits a CBOR sequence into a slice of item slices referencing the original buffer.
func SplitSequenceBytes(b []byte) (out [][]byte, err error) {
	err = ForEachSequenceBytes(b, func(it []byte) error { out = append(out, it); return nil })
	return out, err
}

// AppendSequence appends a sequence of pre-encoded CBOR items to b.
// Each item must be a complete CBOR data item.
func AppendSequence(b []byte, items ...[]byte) []byte {
	for _, it := range items {
		b = append(b, it...)
	}
	return b
}

// AppendSequenceFunc appends n items produced by fn(i), where each returned []byte is a full CBOR item.
func AppendSequenceFunc(b []byte, n int, fn func(i int) ([]byte, error)) ([]byte, error) {
	for i := 0; i < n; i++ {
		it, err := fn(i)
		if err != nil {
			return b, err
		}
		b = append(b, it...)
	}
	return b, nil
}

// ReadOrderedMapBytes reads the next CBOR map (definite or indefinite) and
// returns a slice of RawPair in the order they appeared on the wire.
// Each Key and Value contains exactly one CBOR item (copied).
func ReadOrderedMapBytes(b []byte) (pairs []RawPair, o []byte, err error) {
	if len(b) < 1 {
		return nil, b, ErrShortBytes
	}
	if getMajorType(b[0]) != majorTypeMap {
		return nil, b, badPrefix(majorTypeMap, getMajorType(b[0]))
	}
	// Indefinite-length map
	if getAddInfo(b[0]) == addInfoIndefinite {
		p := b[1:]
		var scratch []byte
		for {
			if len(p) < 1 {
				return nil, b, ErrShortBytes
			}
			if p[0] == makeByte(majorTypeSimple, simpleBreak) {
				return pairs, p[1:], nil
			}
			// Capture raw key
			r1, err := Skip(p)
			if err != nil {
				return nil, b, err
			}
			klen := len(p) - len(r1)
			// Append key bytes into a shared scratch buffer and take a subslice
			startK := len(scratch)
			scratch = append(scratch, p[:klen]...)
			kraw := scratch[startK:]
			// Capture raw value
			r2, err := Skip(r1)
			if err != nil {
				return nil, b, err
			}
			vlen := len(r1) - len(r2)
			startV := len(scratch)
			scratch = append(scratch, r1[:vlen]...)
			vraw := scratch[startV:]
			pairs = append(pairs, RawPair{Key: kraw, Value: vraw})
			p = r2
		}
	}
	// Definite-length map
	sz, p, err := ReadMapHeaderBytes(b)
	if err != nil {
		return nil, b, err
	}
	pairs = make([]RawPair, 0, sz)
	var scratch []byte
	for i := uint32(0); i < sz; i++ {
		r1, err := Skip(p)
		if err != nil {
			return nil, b, err
		}
		klen := len(p) - len(r1)
		startK := len(scratch)
		scratch = append(scratch, p[:klen]...)
		kraw := scratch[startK:]
		r2, err := Skip(r1)
		if err != nil {
			return nil, b, err
		}
		vlen := len(r1) - len(r2)
		startV := len(scratch)
		scratch = append(scratch, r1[:vlen]...)
		vraw := scratch[startV:]
		pairs = append(pairs, RawPair{Key: kraw, Value: vraw})
		p = r2
	}
	return pairs, p, nil
}

// float16BitsToFloat32 converts IEEE 754 binary16 bits to float32
func float16BitsToFloat32(h uint16) float32 {
	sign := uint32(h>>15) & 0x1
	exp := (h >> 10) & 0x1F
	mant := uint32(h & 0x03FF)
	var bits uint32
	switch exp {
	case 0:
		if mant == 0 {
			bits = sign << 31
		} else {
			// subnormal: value = mant / 2^10 * 2^-14 = mant * 2^-24
			// Build float by arithmetic
			f := math.Ldexp(float64(mant), -24)
			if sign != 0 {
				f = -f
			}
			return float32(f)
		}
	case 0x1F:
		// Inf/NaN
		bits = (sign << 31) | (0xFF << 23)
		if mant != 0 {
			bits |= (mant << 13)
		}
	default:
		// normalized
		e32 := int(exp) - 15 + 127
		bits = (sign << 31) | (uint32(e32) << 23) | (mant << 13)
	}
	return math.Float32frombits(bits)
}

// ReadDurationBytes reads a time.Duration
func ReadDurationBytes(b []byte) (d time.Duration, o []byte, err error) {
	i64, o, err := ReadInt64Bytes(b)
	if err != nil {
		return 0, b, err
	}
	return time.Duration(i64), o, nil
}

// ReadMapStrStrBytes reads a map[string]string
func ReadMapStrStrBytes(b []byte, m map[string]string) (o []byte, err error) {
	sz, o, err := ReadMapHeaderBytes(b)
	if err != nil {
		return b, err
	}

	for i := uint32(0); i < sz; i++ {
		var key, val string
		key, o, err = ReadStringBytes(o)
		if err != nil {
			return b, err
		}
		val, o, err = ReadStringBytes(o)
		if err != nil {
			return b, err
		}
		m[key] = val
	}
	return o, nil
}

// Skip skips over the next CBOR object
func Skip(b []byte) ([]byte, error) {
	return skip(b, 0)
}

func skip(b []byte, depth int) ([]byte, error) {
	if depth > recursionLimit {
		return b, ErrMaxDepthExceeded
	}
	if len(b) < 1 {
		return b, ErrShortBytes
	}

	major := getMajorType(b[0])
	addInfo := getAddInfo(b[0])

	switch major {
	case majorTypeUint, majorTypeNegInt, majorTypeTag:
		_, o, err := readUintCore(b, major)
		if err != nil {
			return b, err
		}
		if major == majorTypeTag {
			return skip(o, depth+1)
		}
		return o, nil

	case majorTypeBytes, majorTypeText:
		if addInfo == addInfoIndefinite {
			// Indefinite-length string: series of definite chunks terminated by break
			o := b[1:]
			for {
				if len(o) < 1 {
					return b, ErrShortBytes
				}
				if o[0] == makeByte(majorTypeSimple, simpleBreak) {
					return o[1:], nil
				}
				// Next must be a definite-length chunk of same major type
				sz, q, err := readUintCore(o, major)
				if err != nil {
					return b, err
				}
				if uint64(len(q)) < sz {
					return b, ErrShortBytes
				}
				o = q[sz:]
			}
		}
		sz, o, err := readUintCore(b, major)
		if err != nil {
			return b, err
		}
		if uint64(len(o)) < sz {
			return b, ErrShortBytes
		}
		return o[sz:], nil

	case majorTypeArray:
		if addInfo == addInfoIndefinite {
			o := b[1:]
			for {
				if len(o) < 1 {
					return b, ErrShortBytes
				}
				if o[0] == makeByte(majorTypeSimple, simpleBreak) {
					return o[1:], nil
				}
				var err error
				o, err = skip(o, depth+1)
				if err != nil {
					return b, err
				}
			}
		}
		sz, o, err := readUintCore(b, major)
		if err != nil {
			return b, err
		}
		for i := uint64(0); i < sz; i++ {
			o, err = skip(o, depth+1)
			if err != nil {
				return b, err
			}
		}
		return o, nil

	case majorTypeMap:
		if addInfo == addInfoIndefinite {
			o := b[1:]
			for {
				if len(o) < 1 {
					return b, ErrShortBytes
				}
				if o[0] == makeByte(majorTypeSimple, simpleBreak) {
					return o[1:], nil
				}
				var err error
				o, err = skip(o, depth+1) // key
				if err != nil {
					return b, err
				}
				o, err = skip(o, depth+1) // value
				if err != nil {
					return b, err
				}
			}
		}
		sz, o, err := readUintCore(b, major)
		if err != nil {
			return b, err
		}
		for i := uint64(0); i < sz; i++ {
			o, err = skip(o, depth+1) // key
			if err != nil {
				return b, err
			}
			o, err = skip(o, depth+1) // value
			if err != nil {
				return b, err
			}
		}
		return o, nil

	case majorTypeSimple:
		switch addInfo {
		case simpleFalse, simpleTrue, simpleNull, simpleUndefined:
			return b[1:], nil
		case simpleFloat16:
			if len(b) < 3 {
				return b, ErrShortBytes
			}
			return b[3:], nil
		case simpleFloat32:
			if len(b) < 5 {
				return b, ErrShortBytes
			}
			return b[5:], nil
		case simpleFloat64:
			if len(b) < 9 {
				return b, ErrShortBytes
			}
			return b[9:], nil
		default:
			if addInfo < 20 {
				return b[1:], nil
			}
			return b, &ErrUnsupportedType{}
		}
	}

	return b, &ErrUnsupportedType{}
}

// IsNil checks if the next value is nil
func IsNil(b []byte) bool {
	return len(b) > 0 && b[0] == makeByte(majorTypeSimple, simpleNull)
}

// Raw is raw CBOR data
type Raw []byte

// MarshalCBOR implements Marshaler
func (r Raw) MarshalCBOR(b []byte) ([]byte, error) {
	if len(r) == 0 {
		return AppendNil(b), nil
	}
	return append(b, r...), nil
}

// UnmarshalCBOR implements Unmarshaler
func (r *Raw) UnmarshalCBOR(b []byte) ([]byte, error) {
	l := len(b)
	out, err := Skip(b)
	if err != nil {
		return b, err
	}
	rlen := l - len(out)
	if IsNil(b[:rlen]) {
		rlen = 0
	}
	if cap(*r) < rlen {
		*r = make(Raw, rlen)
	} else {
		*r = (*r)[0:rlen]
	}
	copy(*r, b[:rlen])
	return out, nil
}
