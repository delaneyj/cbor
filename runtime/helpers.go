package cbor

import "unicode/utf8"

// getType returns the CBOR type from a byte
func getType(b byte) Type {
	major := getMajorType(b)
	switch major {
	case majorTypeUint:
		return UintType
	case majorTypeNegInt:
		return IntType
	case majorTypeBytes:
		return BinType
	case majorTypeText:
		return StrType
	case majorTypeArray:
		return ArrayType
	case majorTypeMap:
		return MapType
	case majorTypeTag:
		return ExtensionType
	case majorTypeSimple:
		addInfo := getAddInfo(b)
		switch addInfo {
		case simpleTrue, simpleFalse:
			return BoolType
		case simpleNull:
			return NilType
		case simpleFloat32, simpleFloat64:
			return Float64Type
		}
	}
	return InvalidType
}

// NextType returns the type of the next object in the slice
func NextType(b []byte) Type {
	if len(b) == 0 {
		return InvalidType
	}
	return getType(b[0])
}

// Require ensures that b has capacity for at least n additional bytes
// without reallocation. It returns a slice that shares the original
// contents and has sufficient capacity for appending n bytes.
func Require(b []byte, n int) []byte {
	if cap(b)-len(b) >= n {
		return b
	}
	nb := make([]byte, len(b), len(b)+n)
	copy(nb, b)
	return nb
}

// IsLikelyJSON reports whether the given byte slice looks like JSON text
// rather than CBOR. It is a heuristic and not a formal discriminator:
//
//   - It requires the data to be valid UTF-8.
//   - It then checks the first non-whitespace byte against the JSON
//     value grammar (object/array/string/number/true/false/null).
//
// Most CBOR payloads will fail one of these checks (non-UTF-8 or
// invalid JSON starter) and thus be classified as non-JSON.
func IsLikelyJSON(b []byte) bool {
	// Require valid UTF-8 for JSON.
	if !utf8.Valid(b) {
		return false
	}
	// Skip leading ASCII whitespace.
	i := 0
	for i < len(b) {
		c := b[i]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			i++
			continue
		}
		break
	}
	if i >= len(b) {
		return false
	}
	ch := b[i]
	// Valid JSON value starters:
	//  - object/array: '{', '['
	//  - string: '"'
	//  - number: '-', '0'..'9'
	//  - true/false/null: 't', 'f', 'n'
	if ch == '{' || ch == '[' || ch == '"' || ch == '-' {
		return true
	}
	if ch >= '0' && ch <= '9' {
		return true
	}
	if ch == 't' || ch == 'f' || ch == 'n' {
		return true
	}
	return false
}
