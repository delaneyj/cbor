package cbor

import (
	"math"
	"math/bits"
	"strconv"
)

// Number represents a CBOR number that may be an int64, uint64,
// float32, or float64 internally. The zero value is equivalent to
// an int64 value of 0.
type Number struct {
	bits uint64
	typ  Type
}

// AsInt sets the number to an int64.
func (n *Number) AsInt(i int64) {
	if i == 0 {
		n.typ = InvalidType
		n.bits = 0
		return
	}

	n.typ = IntType
	n.bits = uint64(i)
}

// AsUint sets the number to a uint64.
func (n *Number) AsUint(u uint64) {
	n.typ = UintType
	n.bits = u
}

// AsFloat32 sets the value of the number to a float32.
func (n *Number) AsFloat32(f float32) {
	n.typ = Float32Type
	n.bits = uint64(math.Float32bits(f))
}

// AsFloat64 sets the value of the number to a float64.
func (n *Number) AsFloat64(f float64) {
	n.typ = Float64Type
	n.bits = math.Float64bits(f)
}

// Int returns the value as an int64 and reports whether that was the
// underlying type (or the zero value).
func (n *Number) Int() (int64, bool) {
	return int64(n.bits), n.typ == IntType || n.typ == InvalidType
}

// Uint returns the value as a uint64 and reports whether that was the
// underlying type.
func (n *Number) Uint() (uint64, bool) {
	return n.bits, n.typ == UintType
}

// Float returns the value as a float64 and reports whether the
// underlying type was float32 or float64.
func (n *Number) Float() (float64, bool) {
	switch n.typ {
	case Float32Type:
		return float64(math.Float32frombits(uint32(n.bits))), true
	case Float64Type:
		return math.Float64frombits(n.bits), true
	default:
		return 0, false
	}
}

// Type returns the underlying numeric kind.
func (n *Number) Type() Type {
	if n.typ == InvalidType {
		return IntType
	}
	return n.typ
}

// UnmarshalCBOR decodes a single CBOR number from b into n.
func (n *Number) UnmarshalCBOR(b []byte) ([]byte, error) {
	typ := NextType(b)
	switch typ {
	case IntType:
		i, o, err := ReadInt64Bytes(b)
		if err != nil {
			return b, err
		}
		n.AsInt(i)
		return o, nil
	case UintType:
		u, o, err := ReadUint64Bytes(b)
		if err != nil {
			return b, err
		}
		n.AsUint(u)
		return o, nil
	case Float64Type:
		f, o, err := ReadFloat64Bytes(b)
		if err != nil {
			return b, err
		}
		n.AsFloat64(f)
		return o, nil
	case Float32Type:
		f, o, err := ReadFloat32Bytes(b)
		if err != nil {
			return b, err
		}
		n.AsFloat32(f)
		return o, nil
	default:
		return b, &ErrUnsupportedType{}
	}
}

// MarshalCBOR encodes the stored numeric value into b.
func (n *Number) MarshalCBOR(b []byte) ([]byte, error) {
	switch n.typ {
	case IntType:
		return AppendInt64(b, int64(n.bits)), nil
	case UintType:
		return AppendUint64(b, n.bits), nil
	case Float64Type:
		return AppendFloat64(b, math.Float64frombits(n.bits)), nil
	case Float32Type:
		return AppendFloat32(b, math.Float32frombits(uint32(n.bits))), nil
	default:
		return AppendInt64(b, 0), nil
	}
}

// CoerceInt attempts to coerce the value into an int64 without loss of
// precision and reports success.
func (n *Number) CoerceInt() (int64, bool) {
	switch n.typ {
	case InvalidType, IntType:
		return int64(n.bits), true
	case UintType:
		return int64(n.bits), n.bits <= math.MaxInt64
	case Float32Type:
		f := math.Float32frombits(uint32(n.bits))
		if n.isExactInt() && f <= math.MaxInt64 && f >= math.MinInt64 {
			return int64(f), true
		}
		if n.bits == 0 || n.bits == 1<<31 {
			return 0, true
		}
	case Float64Type:
		f := math.Float64frombits(n.bits)
		if n.isExactInt() && f <= math.MaxInt64 && f >= math.MinInt64 {
			return int64(f), true
		}
		return 0, n.bits == 0 || n.bits == 1<<63
	}
	return 0, false
}

// CoerceUInt attempts to coerce the value into a uint64 without loss of
// precision and reports success.
func (n *Number) CoerceUInt() (uint64, bool) {
	switch n.typ {
	case InvalidType, IntType:
		if int64(n.bits) >= 0 {
			return n.bits, true
		}
	case UintType:
		return n.bits, true
	case Float32Type:
		f := math.Float32frombits(uint32(n.bits))
		if f >= 0 && f <= math.MaxUint64 && n.isExactInt() {
			return uint64(f), true
		}
		if n.bits == 0 || n.bits == 1<<31 {
			return 0, true
		}
	case Float64Type:
		f := math.Float64frombits(n.bits)
		if f >= 0 && f <= math.MaxUint64 && n.isExactInt() {
			return uint64(f), true
		}
		return 0, n.bits == 0 || n.bits == 1<<63
	}
	return 0, false
}

// isExactInt reports whether the stored float value is an exact integer.
func (n *Number) isExactInt() bool {
	var eBits, mBits int

	switch n.typ {
	case InvalidType, IntType, UintType:
		return true
	case Float32Type:
		eBits = 8
		mBits = 23
	case Float64Type:
		eBits = 11
		mBits = 52
	default:
		return false
	}

	exp := int(n.bits>>mBits) & ((1 << eBits) - 1)
	mant := n.bits & ((1 << mBits) - 1)
	if exp == 0 && mant == 0 {
		return true
	}

	exp -= (1 << (eBits - 1)) - 1
	if exp < 0 || exp == 1<<(eBits-1) {
		return false
	}
	if exp >= mBits {
		return true
	}
	return bits.TrailingZeros64(mant) >= mBits-exp
}

// CoerceFloat returns the value as a float64.
func (n *Number) CoerceFloat() float64 {
	switch n.typ {
	case IntType:
		return float64(int64(n.bits))
	case UintType:
		return float64(n.bits)
	case Float32Type:
		return float64(math.Float32frombits(uint32(n.bits)))
	case Float64Type:
		return math.Float64frombits(n.bits)
	default:
		return 0
	}
}

// Msgsize returns the worst-case encoded size.
func (n *Number) Msgsize() int {
	switch n.typ {
	case Float32Type:
		return Float32Size
	case Float64Type:
		return Float64Size
	case IntType:
		return Int64Size
	case UintType:
		return Uint64Size
	default:
		return 1
	}
}

// String implements fmt.Stringer-style formatting.
func (n *Number) String() string {
	switch n.typ {
	case InvalidType:
		return "0"
	case Float32Type, Float64Type:
		f, _ := n.Float()
		return strconv.FormatFloat(f, 'f', -1, 64)
	case IntType:
		i, _ := n.Int()
		return strconv.FormatInt(i, 10)
	case UintType:
		u, _ := n.Uint()
		return strconv.FormatUint(u, 10)
	default:
		return "0"
	}
}

