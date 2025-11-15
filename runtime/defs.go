// This package is the support library for the cborp code generator.
//
// This package defines the utilities used by the cborp code generator for encoding and decoding CBOR
// from []byte and io.Reader/io.Writer types. Much of this package is devoted to helping the cborp code
// generator implement the Marshaler/Unmarshaler and Encodable/Decodable interfaces.
//
// This package defines four "families" of functions:
//   - AppendXxxx() appends an object to a []byte in CBOR encoding.
//   - ReadXxxxBytes() reads an object from a []byte and returns the remaining bytes.
//   - (*Writer).WriteXxxx() writes an object to the buffered *Writer type.
//   - (*Reader).ReadXxxx() reads an object from a buffered *Reader type.
//
// Once a type has satisfied the `Encodable` and `Decodable` interfaces,
// it can be written and read from arbitrary `io.Writer`s and `io.Reader`s using
//
//	cbor.Encode(io.Writer, cbor.Encodable)
//
// and
//
//	cbor.Decode(io.Reader, cbor.Decodable)
package cbor

import "errors"

// RawPair represents an already-encoded CBOR key/value pair.
// Key and Value must each contain exactly one CBOR item.
type RawPair struct {
    Key   []byte
    Value []byte
}
const (
    // recursionLimit is the limit of recursive calls.
    // This limits the call depth of dynamic code, like Skip and interface conversions.
    recursionLimit = 100000
)

// ErrNonCanonicalFloat is returned when a float is not encoded in the shortest form (strict mode).
var ErrNonCanonicalFloat = errors.New("cbor: non-canonical float encoding")

// ErrContainerTooLarge is returned when a container length exceeds configured Reader limits.
var ErrContainerTooLarge = errors.New("cbor: container too large")

// CBOR major types (3 bits)
const (
	majorTypeUint   = 0 // unsigned integer
	majorTypeNegInt = 1 // negative integer
	majorTypeBytes  = 2 // byte string
	majorTypeText   = 3 // text string (UTF-8)
	majorTypeArray  = 4 // array
	majorTypeMap    = 5 // map
	majorTypeTag    = 6 // semantic tag
	majorTypeSimple = 7 // float, simple values, break
)

// Additional info values (5 bits)
const (
	// 0-23: literal value
	addInfoDirect     = 23 // max direct value
	addInfoUint8      = 24 // 1-byte uint8 follows
	addInfoUint16     = 25 // 2-byte uint16 follows
	addInfoUint32     = 26 // 4-byte uint32 follows
	addInfoUint64     = 27 // 8-byte uint64 follows
	addInfoIndefinite = 31 // indefinite length (for bytes, text, array, map)
)

// Simple values in major type 7
const (
	simpleFalse     = 20
	simpleTrue      = 21
	simpleNull      = 22
	simpleUndefined = 23
	simpleFloat16   = 25
	simpleFloat32   = 26
	simpleFloat64   = 27
	simpleBreak     = 31
)

// Common CBOR semantic tags
const (
	tagDateTimeString   = 0     // RFC3339 date/time string
	tagEpochDateTime    = 1     // Unix timestamp (int or float)
	tagPosBignum        = 2     // Positive bignum
	tagNegBignum        = 3     // Negative bignum
	tagDecimalFrac      = 4     // Decimal fraction
	tagBigfloat         = 5     // Bigfloat
	tagBase64URL        = 21    // Expected base64url encoding
	tagBase64           = 22    // Expected base64 encoding
	tagBase16           = 23    // Expected base16 encoding
	tagCBOR             = 24    // Embedded CBOR data item
	tagURI              = 32    // URI
	tagBase64URLString  = 33    // base64url
	tagBase64String     = 34    // base64
	tagRegexp           = 35    // Regular expression
	tagMIME             = 36    // MIME message
	tagSelfDescribeCBOR = 55799 // Self-describe CBOR (0xd9d9f7)
)

// makeByte creates a CBOR initial byte from major type and additional info
func makeByte(majorType, addInfo uint8) byte {
	return byte((majorType << 5) | addInfo)
}

// getMajorType extracts the major type from a CBOR initial byte
func getMajorType(b byte) uint8 {
	return (b >> 5) & 0x07
}

// getAddInfo extracts the additional info from a CBOR initial byte
func getAddInfo(b byte) uint8 {
	return b & 0x1f
}

// Type represents CBOR data types
type Type byte

// CBOR Types
const (
	InvalidType Type = iota

	StrType       // text string
	BinType       // byte string
	MapType       // map
	ArrayType     // array
	Float64Type   // float64
	Float32Type   // float32
	BoolType      // bool
	IntType       // signed integer
	UintType      // unsigned integer
	NilType       // nil
	DurationType  // duration (encoded as int64)
	ExtensionType // tagged value
	TimeType      // time (tagged epoch timestamp)
)

// String implements fmt.Stringer
func (t Type) String() string {
	switch t {
	case StrType:
		return "str"
	case BinType:
		return "bin"
	case MapType:
		return "map"
	case ArrayType:
		return "array"
	case Float64Type:
		return "float64"
	case Float32Type:
		return "float32"
	case BoolType:
		return "bool"
	case UintType:
		return "uint"
	case IntType:
		return "int"
	case ExtensionType:
		return "ext"
	case NilType:
		return "nil"
	case TimeType:
		return "time"
	case DurationType:
		return "duration"
	default:
		return "<invalid>"
	}
}

// Marshaler is the interface implemented by types that know how to marshal
// themselves as CBOR. MarshalCBOR appends the marshalled form to the provided
// byte slice, returning the extended slice and any errors encountered.
type Marshaler interface {
	MarshalCBOR([]byte) ([]byte, error)
}

// Unmarshaler is the interface fulfilled by objects that know how to unmarshal
// themselves from CBOR. UnmarshalCBOR unmarshals the object from binary,
// returning any leftover bytes and any errors encountered.
type Unmarshaler interface {
	UnmarshalCBOR([]byte) ([]byte, error)
}

// ValidateUTF8OnDecode controls whether ReadStringBytes validates UTF-8.
// Enabled by default for spec compliance; can be disabled in hot paths.
var ValidateUTF8OnDecode = true

// UnsafeStringDecode controls whether ReadStringBytes converts zero-copy using
// UnsafeString (unsafe) instead of allocating a new string. Disabled by default.
var UnsafeStringDecode = false

// (duplicate removed)
