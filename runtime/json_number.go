package cbor

import (
	"encoding/json"
	"strconv"
)

// ReadJSONNumberBytes reads a CBOR numeric value and returns it as a
// json.Number, along with the remaining input.
func ReadJSONNumberBytes(b []byte) (json.Number, []byte, error) {
	typ := NextType(b)
	switch typ {
	case IntType:
		v, o, err := ReadInt64Bytes(b)
		if err != nil {
			return "", b, err
		}
		return json.Number(strconv.FormatInt(v, 10)), o, nil
	case UintType:
		v, o, err := ReadUint64Bytes(b)
		if err != nil {
			return "", b, err
		}
		return json.Number(strconv.FormatUint(v, 10)), o, nil
	case Float32Type:
		v, o, err := ReadFloat32Bytes(b)
		if err != nil {
			return "", b, err
		}
		return json.Number(strconv.FormatFloat(float64(v), 'f', -1, 64)), o, nil
	case Float64Type:
		v, o, err := ReadFloat64Bytes(b)
		if err != nil {
			return "", b, err
		}
		return json.Number(strconv.FormatFloat(v, 'f', -1, 64)), o, nil
	default:
		return "", b, &ErrUnsupportedType{}
	}
}

