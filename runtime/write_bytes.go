package cbor

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	bigmath "math/big"
	"regexp"
	"sort"
	"time"
	"reflect"
)

// ensure 'sz' extra bytes in 'b' btw len(b) and cap(b)
func ensure(b []byte, sz int) ([]byte, int) {
	l := len(b)
	c := cap(b)
	if c-l < sz {
		o := make([]byte, (2*c)+sz) // exponential growth
		n := copy(o, b)
		return o[:n+sz], n
	}
	return b[:l+sz], l
}

// appendUintCore encodes an unsigned integer with the given major type
func appendUintCore(b []byte, majorType uint8, u uint64) []byte {
	switch {
	case u <= addInfoDirect:
		return append(b, makeByte(majorType, uint8(u)))
	case u <= math.MaxUint8:
		o, n := ensure(b, 2)
		o[n] = makeByte(majorType, addInfoUint8)
		o[n+1] = uint8(u)
		return o
	case u <= math.MaxUint16:
		o, n := ensure(b, 3)
		o[n] = makeByte(majorType, addInfoUint16)
		binary.BigEndian.PutUint16(o[n+1:], uint16(u))
		return o
	case u <= math.MaxUint32:
		o, n := ensure(b, 5)
		o[n] = makeByte(majorType, addInfoUint32)
		binary.BigEndian.PutUint32(o[n+1:], uint32(u))
		return o
	default:
		o, n := ensure(b, 9)
		o[n] = makeByte(majorType, addInfoUint64)
		binary.BigEndian.PutUint64(o[n+1:], u)
		return o
	}
}

// AppendMapHeader appends a map header with the given size
func AppendMapHeader(b []byte, sz uint32) []byte {
	return appendUintCore(b, majorTypeMap, uint64(sz))
}

// AppendArrayHeader appends an array header with the given size
func AppendArrayHeader(b []byte, sz uint32) []byte {
	return appendUintCore(b, majorTypeArray, uint64(sz))
}

// AppendArrayHeaderIndefinite appends an indefinite-length array header (0x9f)
func AppendArrayHeaderIndefinite(b []byte) []byte {
    return append(b, makeByte(majorTypeArray, addInfoIndefinite))
}

// AppendNil appends a nil value
func AppendNil(b []byte) []byte {
	return append(b, makeByte(majorTypeSimple, simpleNull))
}

// AppendUndefined appends an undefined simple value (23)
func AppendUndefined(b []byte) []byte {
    return append(b, makeByte(majorTypeSimple, simpleUndefined))
}

// AppendTextHeaderIndefinite appends an indefinite-length text string header (0x7f)
func AppendTextHeaderIndefinite(b []byte) []byte { return append(b, makeByte(majorTypeText, addInfoIndefinite)) }

// AppendBytesHeaderIndefinite appends an indefinite-length byte string header (0x5f)
func AppendBytesHeaderIndefinite(b []byte) []byte { return append(b, makeByte(majorTypeBytes, addInfoIndefinite)) }

// AppendTextChunk appends a definite-length text string chunk (use within indefinite text)
func AppendTextChunk(b []byte, s string) []byte { return AppendString(b, s) }

// AppendBytesChunk appends a definite-length byte string chunk (use within indefinite bytes)
func AppendBytesChunk(b []byte, bs []byte) []byte { return AppendBytes(b, bs) }

// AppendFloat64 appends a float64
func AppendFloat64(b []byte, f float64) []byte {
	o, n := ensure(b, 9)
	o[n] = makeByte(majorTypeSimple, simpleFloat64)
	binary.BigEndian.PutUint64(o[n+1:], math.Float64bits(f))
	return o
}

// AppendFloat32 appends a float32
func AppendFloat32(b []byte, f float32) []byte {
	o, n := ensure(b, 5)
	o[n] = makeByte(majorTypeSimple, simpleFloat32)
	binary.BigEndian.PutUint32(o[n+1:], math.Float32bits(f))
	return o
}

// AppendFloatCanonical appends the shortest-width float (f16/f32/f64) that preserves the value.
func AppendFloatCanonical(b []byte, f float64) []byte {
    // Normalize -0 to +0 for canonical
    if f == 0 && math.Signbit(f) { f = 0 }
    // NaN: canonicalize to float16 NaN
    if math.IsNaN(f) {
        return AppendFloat16(b, float32(f))
    }
    // Try f16
    f16 := float32ToFloat16Bits(float32(f))
    if float64(float16BitsToFloat32(f16)) == f {
        return AppendFloat16(b, float32(f))
    }
    // Try f32
    f32 := float32(f)
    if float64(f32) == f {
        return AppendFloat32(b, f32)
    }
    return AppendFloat64(b, f)
}

// AppendFloat16 appends a float16 (IEEE 754 binary16) encoded value
func AppendFloat16(b []byte, f float32) []byte {
	o, n := ensure(b, 3)
	o[n] = makeByte(majorTypeSimple, simpleFloat16)
	binary.BigEndian.PutUint16(o[n+1:], float32ToFloat16Bits(f))
	return o
}

// AppendFloat appends a float as float32 if it represents the same value, else float64
func AppendFloat(b []byte, f float64) []byte {
	f32 := float32(f)
	if float64(f32) == f {
		return AppendFloat32(b, f32)
	}
	return AppendFloat64(b, f)
}

// AppendDuration appends a time.Duration as int64
func AppendDuration(b []byte, d time.Duration) []byte {
	return AppendInt64(b, int64(d))
}

// AppendInt64 appends an int64 using canonical CBOR integer encoding.
//
// For small values in the common ranges we specialize the encoding
// inline rather than routing through appendUintCore. This mirrors the
// fast-path treatment used in the original tinylib/msgp runtime while
// preserving CBOR's major-type and additional-info layout.
func AppendInt64(b []byte, i int64) []byte {
	// Fast path for small positive values 0..23 (single-byte encoding).
	if i >= 0 && i <= addInfoDirect {
		return append(b, makeByte(majorTypeUint, uint8(i)))
	}
	// Fast path for small negative values -1..-24. CBOR encodes
	// negative integers as -1-n with unsigned argument n.
	if i < 0 {
		neg := -1 - i // n such that value = -1-n
		if neg >= 0 && neg <= addInfoDirect {
			return append(b, makeByte(majorTypeNegInt, uint8(neg)))
		}
		return appendUintCore(b, majorTypeNegInt, uint64(neg))
	}
	// Remaining positive values go through the generic uint encoder.
	return appendUintCore(b, majorTypeUint, uint64(i))
}

// AppendInt appends an int
func AppendInt(b []byte, i int) []byte {
	return AppendInt64(b, int64(i))
}

// AppendInt8 appends an int8
func AppendInt8(b []byte, i int8) []byte {
	return AppendInt64(b, int64(i))
}

// AppendInt16 appends an int16
func AppendInt16(b []byte, i int16) []byte {
	return AppendInt64(b, int64(i))
}

// AppendInt32 appends an int32
func AppendInt32(b []byte, i int32) []byte {
	return AppendInt64(b, int64(i))
}

// AppendUint64 appends a uint64
func AppendUint64(b []byte, u uint64) []byte {
	return appendUintCore(b, majorTypeUint, u)
}

// AppendUint appends a uint
func AppendUint(b []byte, u uint) []byte {
	return AppendUint64(b, uint64(u))
}

// AppendUint8 appends a uint8
func AppendUint8(b []byte, u uint8) []byte {
	return appendUintCore(b, majorTypeUint, uint64(u))
}

// AppendUint16 appends a uint16
func AppendUint16(b []byte, u uint16) []byte {
	return appendUintCore(b, majorTypeUint, uint64(u))
}

// AppendUint32 appends a uint32
func AppendUint32(b []byte, u uint32) []byte {
	return appendUintCore(b, majorTypeUint, uint64(u))
}

// AppendBytes appends a byte string
func AppendBytes(b []byte, data []byte) []byte {
    sz := uint64(len(data))
    // Compute header size and reserve in one shot to avoid double ensure + copy
    var h int
    switch {
    case sz <= addInfoDirect:
        h = 1
    case sz <= math.MaxUint8:
        h = 2
    case sz <= math.MaxUint16:
        h = 3
    case sz <= math.MaxUint32:
        h = 5
    default:
        h = 9
    }
    o, n := ensure(b, h+int(sz))
    // Write header
    switch h {
    case 1:
        o[n] = makeByte(majorTypeBytes, uint8(sz))
        n++
    case 2:
        o[n] = makeByte(majorTypeBytes, addInfoUint8)
        o[n+1] = uint8(sz)
        n += 2
    case 3:
        o[n] = makeByte(majorTypeBytes, addInfoUint16)
        binary.BigEndian.PutUint16(o[n+1:], uint16(sz))
        n += 3
    case 5:
        o[n] = makeByte(majorTypeBytes, addInfoUint32)
        binary.BigEndian.PutUint32(o[n+1:], uint32(sz))
        n += 5
    case 9:
        o[n] = makeByte(majorTypeBytes, addInfoUint64)
        binary.BigEndian.PutUint64(o[n+1:], sz)
        n += 9
    }
    // Copy payload
    copy(o[n:], data)
    return o[:n+int(sz)]
}

// AppendString appends a text string
func AppendString(b []byte, s string) []byte {
    sz := uint64(len(s))
    // Compute header size and reserve once
    var h int
    switch {
    case sz <= addInfoDirect:
        h = 1
    case sz <= math.MaxUint8:
        h = 2
    case sz <= math.MaxUint16:
        h = 3
    case sz <= math.MaxUint32:
        h = 5
    default:
        h = 9
    }
    o, n := ensure(b, h+int(sz))
    // Write header
    switch h {
    case 1:
        o[n] = makeByte(majorTypeText, uint8(sz))
        n++
    case 2:
        o[n] = makeByte(majorTypeText, addInfoUint8)
        o[n+1] = uint8(sz)
        n += 2
    case 3:
        o[n] = makeByte(majorTypeText, addInfoUint16)
        binary.BigEndian.PutUint16(o[n+1:], uint16(sz))
        n += 3
    case 5:
        o[n] = makeByte(majorTypeText, addInfoUint32)
        binary.BigEndian.PutUint32(o[n+1:], uint32(sz))
        n += 5
    case 9:
        o[n] = makeByte(majorTypeText, addInfoUint64)
        binary.BigEndian.PutUint64(o[n+1:], sz)
        n += 9
    }
    // Copy payload
    copy(o[n:], s)
    return o[:n+int(sz)]
}

// AppendStringFromBytes appends a string from bytes
func AppendStringFromBytes(b []byte, data []byte) []byte {
	sz := uint64(len(data))
	b = appendUintCore(b, majorTypeText, sz)
	return append(b, data...)
}

// AppendBool appends a bool
func AppendBool(b []byte, val bool) []byte {
	if val {
		return append(b, makeByte(majorTypeSimple, simpleTrue))
	}
	return append(b, makeByte(majorTypeSimple, simpleFalse))
}

// AppendSimpleValue appends a generic simple value.
// Values 0..23 are encoded in the additional information;
// values 32..255 are encoded as 0xf8 XX.
// Note: 24..27 are reserved for float encodings and are not handled here.
func AppendSimpleValue(b []byte, val uint8) []byte {
	switch {
	case val <= addInfoDirect:
		return append(b, makeByte(majorTypeSimple, val))
	default:
		o, n := ensure(b, 2)
		o[n] = makeByte(majorTypeSimple, addInfoUint8) // 0xf8
		o[n+1] = val
		return o
	}
}

// AppendTime appends a time.Time as CBOR tag 1 (epoch timestamp)
func AppendTime(b []byte, t time.Time) []byte {
	b = AppendTag(b, tagEpochDateTime)
	sec := t.Unix()
	nsec := t.Nanosecond()
	if nsec == 0 {
		return AppendInt64(b, sec)
	}
	f := float64(sec) + float64(nsec)/1e9
	return AppendFloat64(b, f)
}

// AppendTag appends a generic semantic tag
func AppendTag(b []byte, tag uint64) []byte {
	return appendUintCore(b, majorTypeTag, tag)
}

// AppendTagged appends a tag followed by a pre-encoded value
func AppendTagged(b []byte, tag uint64, value []byte) []byte {
	b = AppendTag(b, tag)
	return append(b, value...)
}

// AppendRFC3339Time appends a tag(0) RFC3339 datetime string
func AppendRFC3339Time(b []byte, t time.Time) []byte {
    b = AppendTag(b, tagDateTimeString)
    return AppendString(b, t.Format(time.RFC3339Nano))
}

// AppendBase64URLString appends tag(33) with a base64url text string payload
func AppendBase64URLString(b []byte, s string) []byte {
    b = AppendTag(b, tagBase64URLString)
    return AppendString(b, s)
}

// AppendBase64String appends tag(34) with a base64 text string payload
func AppendBase64String(b []byte, s string) []byte {
    b = AppendTag(b, tagBase64String)
    return AppendString(b, s)
}

// AppendURI appends a tag(32) URI text string
func AppendURI(b []byte, uri string) []byte {
	b = AppendTag(b, tagURI)
	return AppendString(b, uri)
}

// AppendEmbeddedCBOR appends tag(24) with a byte string containing embedded CBOR payload
func AppendEmbeddedCBOR(b []byte, payload []byte) []byte {
	b = AppendTag(b, tagCBOR)
	return AppendBytes(b, payload)
}

// AppendUUID appends tag(37) with a 16-byte UUID (RFC 4122) as byte string
func AppendUUID(b []byte, uuid [16]byte) []byte {
    b = AppendTag(b, 37)
    return AppendBytes(b, uuid[:])
}

// AppendRegexpString appends tag(35) with a regular expression pattern as text
func AppendRegexpString(b []byte, re string) []byte {
    b = AppendTag(b, tagRegexp)
    return AppendString(b, re)
}

// AppendMIMEString appends tag(36) with a MIME message as text
func AppendMIMEString(b []byte, mime string) []byte {
    b = AppendTag(b, tagMIME)
    return AppendString(b, mime)
}

// AppendSelfDescribeCBOR appends the self-describe CBOR tag (0xd9d9f7)
func AppendSelfDescribeCBOR(b []byte) []byte {
    return appendUintCore(b, majorTypeTag, tagSelfDescribeCBOR)
}

// AppendRegexp appends tag(35) from a compiled *regexp.Regexp
func AppendRegexp(b []byte, re *regexp.Regexp) []byte {
    if re == nil {
        return AppendNil(b)
    }
    return AppendRegexpString(b, re.String())
}

// AppendBigInt appends a big integer using bignum tags (2 positive, 3 negative)
func AppendBigInt(b []byte, z *bigmath.Int) []byte {
	if z == nil {
		return AppendNil(b)
	}
	if z.Sign() >= 0 {
		b = AppendTag(b, tagPosBignum)
		return AppendBytes(b, z.Bytes())
	}
	// Negative: encode n = -1 - value
	tmp := new(bigmath.Int).Neg(z)  // -z
	tmp.Sub(tmp, bigmath.NewInt(1)) // -z - 1
	b = AppendTag(b, tagNegBignum)
	return AppendBytes(b, tmp.Bytes())
}

// appendCBORIntegerFromBigInt encodes a big.Int as the shortest CBOR integer or bignum.
func appendCBORIntegerFromBigInt(b []byte, z *bigmath.Int) []byte {
	if z == nil {
		return AppendNil(b)
	}
	if z.Sign() >= 0 && z.BitLen() <= 64 {
		return AppendUint64(b, z.Uint64())
	}
	if z.Sign() < 0 && z.BitLen() <= 63 {
		return AppendInt64(b, z.Int64())
	}
	return AppendBigInt(b, z)
}

// AppendDecimalFraction appends tag(4) decimal fraction [exponent, mantissa]
func AppendDecimalFraction(b []byte, exponent int64, mantissa *bigmath.Int) []byte {
	b = AppendTag(b, tagDecimalFrac)
	b = AppendArrayHeader(b, 2)
	b = AppendInt64(b, exponent)
	b = appendCBORIntegerFromBigInt(b, mantissa)
	return b
}

// AppendBigfloat appends tag(5) bigfloat [exponent, mantissa]
func AppendBigfloat(b []byte, exponent int64, mantissa *bigmath.Int) []byte {
	b = AppendTag(b, tagBigfloat)
	b = AppendArrayHeader(b, 2)
	b = AppendInt64(b, exponent)
	b = appendCBORIntegerFromBigInt(b, mantissa)
	return b
}

// AppendBase64URL appends tag(21) with a byte string payload
func AppendBase64URL(b []byte, data []byte) []byte {
	b = AppendTag(b, tagBase64URL)
	return AppendBytes(b, data)
}

// AppendBase64 appends tag(22) with a byte string payload
func AppendBase64(b []byte, data []byte) []byte {
	b = AppendTag(b, tagBase64)
	return AppendBytes(b, data)
}

// AppendBase16 appends tag(23) with a byte string payload
func AppendBase16(b []byte, data []byte) []byte {
	b = AppendTag(b, tagBase16)
	return AppendBytes(b, data)
}

// float32ToFloat16Bits converts float32 to IEEE 754 binary16 representation (round to nearest even)
func float32ToFloat16Bits(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := uint16((bits >> 31) & 0x1)
	exp := int((bits >> 23) & 0xFF)
	mant := bits & 0x7FFFFF

	var h uint16
	switch exp {
	case 0xFF: // NaN or Inf
		if mant == 0 {
			h = (0x1F << 10) // Inf
		} else {
			h = (0x1F << 10) | uint16(mant>>13)
			if h&0x03FF == 0 { // ensure NaN payload
				h |= 1
			}
		}
	case 0: // zero or subnormal in f32 => maps to zero in f16 or very small subnormals (flush to zero)
		// Treat as subnormal; result rounds to 0 for f16 granularity
		h = 0
	default:
		// Normalized number
		// Unbias exponent: e32 = exp-127; target e16 = e32 + 15
		e32 := exp - 127
		e16 := e32 + 15
		if e16 >= 0x1F { // overflow => Inf
			h = (0x1F << 10)
		} else if e16 <= 0 { // subnormal or underflow
			// subnormal half: significand = (mant | 1<<23) >> (1 - e16 + 13)
			// shift = 14 - e32 = 14 - (exp-127) = 141 - exp
			shift := 14 - e32
			if shift > 24 { // too small => zero
				h = 0
			} else {
				mantissa := (mant | 1<<23)
				// add rounding bias before shifting to 10 bits
				round := uint32(1) << (shift - 1)
				val := uint32(mantissa)
				val += round - 1 + ((val >> (shift)) & 1) // round to even
				frac := uint16(val >> shift)
				h = frac & 0x03FF
			}
		} else {
			// normal half
			// round mantissa from 23 to 10 bits
			mantR := mant
			round := uint32(1) << 12
			val := mantR + round - 1 + ((mantR >> 13) & 1)
			frac := uint16(val >> 13)
			h = (uint16(e16) << 10) | (frac & 0x03FF)
			if frac>>10 != 0 { // mantissa overflow rounded up exponent
				// carry into exponent
				e16++
				if e16 >= 0x1F {
					h = (0x1F << 10)
				} else {
					h = (uint16(e16) << 10)
				}
			}
		}
	}
	return (sign << 15) | h
}

// AppendMapStrStr appends a map[string]string
func AppendMapStrStr(b []byte, m map[string]string) []byte {
	sz := uint32(len(m))
	b = AppendMapHeader(b, sz)
	for key, val := range m {
		b = AppendString(b, key)
		b = AppendString(b, val)
	}
	return b
}

// AppendMapStrInterface appends a map[string]any
func AppendMapStrInterface(b []byte, m map[string]any) ([]byte, error) {
	sz := uint32(len(m))
	b = AppendMapHeader(b, sz)
	for key, val := range m {
		b = AppendString(b, key)
		var err error
		b, err = AppendInterface(b, val)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

// AppendStringSlice appends a []string as a CBOR array of text strings.
func AppendStringSlice(b []byte, v []string) []byte {
	b = AppendArrayHeader(b, uint32(len(v)))
	for _, s := range v {
		b = AppendString(b, s)
	}
	return b
}

// AppendMapUint64Marshaler appends map[uint64]T to a CBOR map, where T has
// a corresponding Marshaler implementation (either as value or pointer).
// This is intended for generated code to avoid dynamic map handling in
// AppendInterface for map[uint64]*Struct shapes like ConsumerState.Pending.
func AppendMapUint64Marshaler[T any](b []byte, m map[uint64]T) ([]byte, error) {
	b = AppendMapHeader(b, uint32(len(m)))
	var err error
	for k, v := range m {
		b = AppendUint64(b, k)
		var mval Marshaler
		if mm, ok := any(v).(Marshaler); ok {
			mval = mm
		} else if mm, ok := any(&v).(Marshaler); ok {
			mval = mm
		} else {
			return b, &ErrUnsupportedType{}
		}
		b, err = mval.MarshalCBOR(b)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

// AppendMapUint64Uint64 appends a map[uint64]uint64 as a CBOR map with
// uint64 keys and values. This is used for ConsumerState.Redelivered and
// avoids reflection or interface-based encoding.
func AppendMapUint64Uint64(b []byte, m map[uint64]uint64) []byte {
	b = AppendMapHeader(b, uint32(len(m)))
	for k, v := range m {
		b = AppendUint64(b, k)
		b = AppendUint64(b, v)
	}
	return b
}

// AppendPtrMarshaler appends a pointer to a value that implements
// Marshaler. If the pointer is nil, a CBOR null is written. This is
// primarily for generated code (cborgen) to avoid the generic
// AppendInterface path for pointer-to-struct fields.
func AppendPtrMarshaler[T any](b []byte, v *T) ([]byte, error) {
	if v == nil {
		return AppendNil(b), nil
	}
	if m, ok := any(v).(Marshaler); ok {
		return m.MarshalCBOR(b)
	}
	return b, &ErrUnsupportedType{}
}

// AppendSliceMarshaler appends a slice of values that have a corresponding
// Marshaler implementation to a CBOR array. It is intended for use by
// generated code (cborgen) to avoid per-element AppendInterface overhead.
//
// T may be a type that itself implements Marshaler or whose pointer type
// implements Marshaler (the common case for generated methods).
func AppendSliceMarshaler[T any](b []byte, v []T) ([]byte, error) {
	b = AppendArrayHeader(b, uint32(len(v)))
	var err error
	for i := range v {
		var m Marshaler
		if mm, ok := any(v[i]).(Marshaler); ok {
			m = mm
		} else if mm, ok := any(&v[i]).(Marshaler); ok {
			m = mm
		} else {
			return b, &ErrUnsupportedType{}
		}
		b, err = m.MarshalCBOR(b)
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

// AppendInterface appends an arbitrary value
func AppendInterface(b []byte, i any) ([]byte, error) {
	if i == nil {
		return AppendNil(b), nil
	}

	switch v := i.(type) {
	case Marshaler:
		return v.MarshalCBOR(b)
	case string:
		return AppendString(b, v), nil
	case bool:
		return AppendBool(b, v), nil
	case int:
		return AppendInt(b, v), nil
	case int8:
		return AppendInt8(b, v), nil
	case int16:
		return AppendInt16(b, v), nil
	case int32:
		return AppendInt32(b, v), nil
	case int64:
		return AppendInt64(b, v), nil
	case uint:
		return AppendUint(b, v), nil
	case uint8:
		return AppendUint8(b, v), nil
	case uint16:
		return AppendUint16(b, v), nil
	case uint32:
		return AppendUint32(b, v), nil
	case uint64:
		return AppendUint64(b, v), nil
	case float32:
		return AppendFloat32(b, v), nil
	case float64:
		return AppendFloat64(b, v), nil
	case []byte:
		return AppendBytes(b, v), nil
	case time.Time:
		return AppendTime(b, v), nil
	case time.Duration:
		return AppendDuration(b, v), nil
	case []int:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendInt(b, elem) }
		return b, nil
	case []int8:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendInt8(b, elem) }
		return b, nil
	case []int16:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendInt16(b, elem) }
		return b, nil
	case []int32:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendInt32(b, elem) }
		return b, nil
	case []int64:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendInt64(b, elem) }
		return b, nil
	case []uint:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendUint(b, elem) }
		return b, nil
	case []uint16:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendUint16(b, elem) }
		return b, nil
	case []uint32:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendUint32(b, elem) }
		return b, nil
	case []uint64:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendUint64(b, elem) }
		return b, nil
	case []float32:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendFloat32(b, elem) }
		return b, nil
	case []float64:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendFloat64(b, elem) }
		return b, nil
	case []string:
		b = AppendArrayHeader(b, uint32(len(v)))
		for _, elem := range v { b = AppendString(b, elem) }
		return b, nil
	case map[string]int:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendInt(b, val)
		}
		return b, nil
	case map[string]int64:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendInt64(b, val)
		}
		return b, nil
	case map[string]uint:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendUint(b, val)
		}
		return b, nil
	case map[string]uint64:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendUint64(b, val)
		}
		return b, nil
	case map[string]float64:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendFloat64(b, val)
		}
		return b, nil
	case map[string]string:
		b = AppendMapHeader(b, uint32(len(v)))
		for k, val := range v {
			b = AppendString(b, k)
			b = AppendString(b, val)
		}
		return b, nil
	case json.RawMessage:
		// Treat RawMessage as an opaque CBOR byte string.
		return AppendBytes(b, []byte(v)), nil
	case json.Number:
		if iv, err := v.Int64(); err == nil {
			return AppendInt64(b, iv), nil
		}
		if fv, err := v.Float64(); err == nil {
			return AppendFloat64(b, fv), nil
		}
		return b, &ErrUnsupportedType{}
	case map[string]any:
		return AppendMapStrInterface(b, v)
	case []any:
		b = AppendArrayHeader(b, uint32(len(v)))
		var err error
		for _, elem := range v {
			b, err = AppendInterface(b, elem)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	default:
		// Fallback: handle slices and maps of Marshaler types via reflection.
		rv := reflect.ValueOf(i)
		t := rv.Type()
		if rv.Kind() == reflect.Slice {
			b = AppendArrayHeader(b, uint32(rv.Len()))
			for idx := 0; idx < rv.Len(); idx++ {
				val := rv.Index(idx)
				elem := val.Interface()
				m, ok := elem.(Marshaler)
				if !ok && val.CanAddr() {
					m, ok = val.Addr().Interface().(Marshaler)
				}
				if !ok {
					return b, &ErrUnsupportedType{}
				}
				var err error
				b, err = m.MarshalCBOR(b)
				if err != nil {
					return b, err
				}
			}
			return b, nil
		}
		if rv.Kind() == reflect.Map {
			keyKind := t.Key().Kind()
			keys := rv.MapKeys()
			b = AppendMapHeader(b, uint32(len(keys)))
			for _, k := range keys {
				// Encode the key according to its kind.
				switch keyKind {
				case reflect.String:
					b = AppendString(b, k.String())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					b = AppendUint64(b, k.Uint())
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					b = AppendInt64(b, k.Int())
				default:
					return b, &ErrUnsupportedType{}
				}

				mv := rv.MapIndex(k)
				val := mv.Interface()
				// Prefer Marshaler if available. For common generated
				// patterns that use pointer receivers on value fields,
				// synthesize a pointer and try that before falling back
				// to AppendInterface.
				if m, ok := val.(Marshaler); ok {
					var err error
					b, err = m.MarshalCBOR(b)
					if err != nil {
						return b, err
					}
				} else {
					// Try pointer to value type.
					ptr := reflect.New(mv.Type())
					ptr.Elem().Set(mv)
					if m, ok := ptr.Interface().(Marshaler); ok {
						var err error
						b, err = m.MarshalCBOR(b)
						if err != nil {
							return b, err
						}
						continue
					}

					var err error
					b, err = AppendInterface(b, val)
					if err != nil {
						return b, err
					}
				}
			}
			return b, nil
		}
		return b, &ErrUnsupportedType{}
	}
}

// AppendMapStrStrDeterministic appends a map[string]string with keys sorted by encoded key bytes.
func AppendMapStrStrDeterministic(b []byte, m map[string]string) []byte {
	sz := uint32(len(m))
	b = AppendMapHeader(b, sz)
	type kv struct {
		key string
		enc []byte
	}
	arr := make([]kv, 0, len(m))
	for k := range m {
		arr = append(arr, kv{key: k, enc: AppendString(nil, k)})
	}
	sort.Slice(arr, func(i, j int) bool { return bytes.Compare(arr[i].enc, arr[j].enc) < 0 })
	for _, it := range arr {
		b = AppendString(b, it.key)
		b = AppendString(b, m[it.key])
	}
	return b
}

// AppendMapStrInterfaceDeterministic appends a map[string]any with keys sorted by encoded key bytes.
func AppendMapStrInterfaceDeterministic(b []byte, m map[string]any) ([]byte, error) {
	sz := uint32(len(m))
	b = AppendMapHeader(b, sz)
	type kv struct {
		key string
		enc []byte
	}
	arr := make([]kv, 0, len(m))
	for k := range m {
		arr = append(arr, kv{key: k, enc: AppendString(nil, k)})
	}
	sort.Slice(arr, func(i, j int) bool { return bytes.Compare(arr[i].enc, arr[j].enc) < 0 })
	for _, it := range arr {
		b = AppendString(b, it.key)
		var err error
		b, err = AppendInterface(b, m[it.key])
		if err != nil {
			return b, err
		}
	}
	return b, nil
}

// AppendMapHeaderIndefinite appends an indefinite-length map header (0xbf)
func AppendMapHeaderIndefinite(b []byte) []byte {
	return append(b, makeByte(majorTypeMap, addInfoIndefinite))
}

// AppendBreak appends a break stop code (0xff)
func AppendBreak(b []byte) []byte {
    return append(b, makeByte(majorTypeSimple, simpleBreak))
}

// AppendRawMapDeterministic appends a map with entries provided as raw CBOR key/value pairs.
// Pairs are sorted by CBOR-encoded key bytes to ensure RFC 8949 deterministic order.
func AppendRawMapDeterministic(b []byte, pairs []RawPair) []byte {
    // Deterministic order: by encoded key length, then bytewise lexicographic.
    n := len(pairs)
    if n == 0 {
        return AppendMapHeader(b, 0)
    }
    // Bucket indices by key length.
    byLen := make(map[int][]int)
    for i := 0; i < n; i++ {
        l := len(pairs[i].Key)
        byLen[l] = append(byLen[l], i)
    }
    lens := make([]int, 0, len(byLen))
    for l := range byLen { lens = append(lens, l) }
    sort.Ints(lens)
    order := make([]int, 0, n)
    counts := make([]int, 256)
    var tmp []int
    for _, l := range lens {
        grp := byLen[l]
        if len(grp) <= 1 {
            order = append(order, grp...)
            continue
        }
        // Adaptive: comparator is faster for smaller groups/short keys.
        if l < 64 && len(grp) < 1024 {
            sort.Slice(grp, func(i, j int) bool { return bytes.Compare(pairs[grp[i]].Key, pairs[grp[j]].Key) < 0 })
            order = append(order, grp...)
            continue
        }
        if cap(tmp) < len(grp) { tmp = make([]int, len(grp)) } else { tmp = tmp[:len(grp)] }
        cur := grp
        aux := tmp
        for pos := l - 1; pos >= 0; pos-- {
            for i := range counts { counts[i] = 0 }
            for _, idx := range cur { counts[int(pairs[idx].Key[pos])]++ }
            sum := 0
            for i := 0; i < 256; i++ { c := counts[i]; counts[i] = sum; sum += c }
            for _, idx := range cur {
                bv := pairs[idx].Key[pos]
                p := counts[int(bv)]
                aux[p] = idx
                counts[int(bv)] = p + 1
            }
            cur, aux = aux, cur
        }
        order = append(order, cur...)
    }
    b = AppendMapHeader(b, uint32(n))
    for _, i := range order {
        b = append(b, pairs[i].Key...)
        b = append(b, pairs[i].Value...)
    }
    return b
}

// AppendMapDeterministic appends a map[K]V deterministically.
// encKey appends the CBOR encoding of key k to dst and returns the extended dst.
// encVal appends the CBOR encoding of value v to dst and returns the extended dst.
// Keys are encoded once for sorting and then reused to avoid re-encoding.
func AppendMapDeterministic[K comparable, V any](b []byte, m map[K]V,
    encKey func(dst []byte, k K) []byte,
    encVal func(dst []byte, v V) ([]byte, error),
) ([]byte, error) {
    type item struct {
        keyEnc []byte
        key    K
        val    V
    }
    items := make([]item, 0, len(m))
    // Use a single growing scratch buffer to hold all encoded keys.
    // This reduces per-key allocations. Each keyEnc stores a subslice
    // of the scratch at the moment of encoding; later growth may reallocate
    // scratch, but the subslices retain references to the older backing arrays.
    var scratch []byte
    for k, v := range m {
        prev := len(scratch)
        scratch = encKey(scratch, k)
        ke := scratch[prev:]
        items = append(items, item{keyEnc: ke, key: k, val: v})
    }
    // Bucket by encoded key length, then LSD radix within groups.
    byLen := make(map[int][]int)
    for i := range items { byLen[len(items[i].keyEnc)] = append(byLen[len(items[i].keyEnc)], i) }
    lens := make([]int, 0, len(byLen))
    for l := range byLen { lens = append(lens, l) }
    sort.Ints(lens)
    order := make([]int, 0, len(items))
    counts := make([]int, 256)
    var tmpIdx []int
    for _, l := range lens {
        grp := byLen[l]
        if len(grp) <= 1 { order = append(order, grp...); continue }
        // Adaptive: comparator wins for small groups/short keys
        if l < 64 && len(grp) < 1024 {
            sort.Slice(grp, func(i, j int) bool { return bytes.Compare(items[grp[i]].keyEnc, items[grp[j]].keyEnc) < 0 })
            order = append(order, grp...)
            continue
        }
        if cap(tmpIdx) < len(grp) { tmpIdx = make([]int, len(grp)) } else { tmpIdx = tmpIdx[:len(grp)] }
        cur := grp
        aux := tmpIdx
        for pos := l - 1; pos >= 0; pos-- {
            for i := range counts { counts[i] = 0 }
            for _, idx := range cur { counts[int(items[idx].keyEnc[pos])]++ }
            sum := 0
            for i := 0; i < 256; i++ { c := counts[i]; counts[i] = sum; sum += c }
            for _, idx := range cur {
                bv := items[idx].keyEnc[pos]
                p := counts[int(bv)]
                aux[p] = idx
                counts[int(bv)] = p + 1
            }
            cur, aux = aux, cur
        }
        order = append(order, cur...)
    }
    b = AppendMapHeader(b, uint32(len(items)))
    var err error
    for _, oi := range order {
        b = append(b, items[oi].keyEnc...)
        b, err = encVal(b, items[oi].val)
        if err != nil { return b, err }
    }
    return b, nil
}

// Common key encoders (for AppendMapDeterministic)
func EncKeyString(dst []byte, s string) []byte  { return AppendString(dst, s) }
func EncKeyBytes(dst []byte, bs []byte) []byte  { return AppendBytes(dst, bs) }
func EncKeyInt(dst []byte, i int) []byte        { return AppendInt(dst, i) }
func EncKeyInt64(dst []byte, i int64) []byte    { return AppendInt64(dst, i) }
func EncKeyUint64(dst []byte, u uint64) []byte  { return AppendUint64(dst, u) }
func EncKeyBool(dst []byte, v bool) []byte      { return AppendBool(dst, v) }
func EncKeyFloat64(dst []byte, f float64) []byte{ return AppendFloat64(dst, f) }
func EncKeyTime(dst []byte, t time.Time) []byte { return AppendTime(dst, t) }

// Common value encoders (for AppendMapDeterministic)
func EncValString(dst []byte, s string) ([]byte, error)   { return AppendString(dst, s), nil }
func EncValBytes(dst []byte, bs []byte) ([]byte, error)   { return AppendBytes(dst, bs), nil }
func EncValInt(dst []byte, i int) ([]byte, error)         { return AppendInt(dst, i), nil }
func EncValInt64(dst []byte, i int64) ([]byte, error)     { return AppendInt64(dst, i), nil }
func EncValUint64(dst []byte, u uint64) ([]byte, error)   { return AppendUint64(dst, u), nil }
func EncValBool(dst []byte, v bool) ([]byte, error)       { return AppendBool(dst, v), nil }
func EncValFloat64(dst []byte, f float64) ([]byte, error) { return AppendFloat64(dst, f), nil }
func EncValFloat32(dst []byte, f float32) ([]byte, error) { return AppendFloat32(dst, f), nil }
func EncValTime(dst []byte, t time.Time) ([]byte, error)  { return AppendTime(dst, t), nil }

// EncValInterface appends an arbitrary value.
func EncValInterface(dst []byte, v any) ([]byte, error) { return AppendInterface(dst, v) }

// Typed deterministic appenders for common key/value types
func AppendMapDeterministicStrStr(b []byte, m map[string]string) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValString)
    return out
}

func AppendMapDeterministicStrInt64(b []byte, m map[string]int64) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValInt64)
    return out
}

func AppendMapDeterministicStrInt(b []byte, m map[string]int) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValInt)
    return out
}

func AppendMapDeterministicStrUint64(b []byte, m map[string]uint64) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValUint64)
    return out
}

func AppendMapDeterministicStrBool(b []byte, m map[string]bool) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValBool)
    return out
}

func AppendMapDeterministicStrFloat64(b []byte, m map[string]float64) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValFloat64)
    return out
}

func AppendMapDeterministicStrBytes(b []byte, m map[string][]byte) []byte {
    out, _ := AppendMapDeterministic(b, m, EncKeyString, EncValBytes)
    return out
}

func AppendMapDeterministicStrInterface(b []byte, m map[string]any) ([]byte, error) {
	return AppendMapDeterministic(b, m, EncKeyString, EncValInterface)
}
