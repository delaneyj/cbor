package cbor

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"regexp"
	"strings"
	"time"
)

// FromJSONBytes converts a JSON document into CBOR bytes using a
// wrapper convention similar to RFC examples and the prototype:
//
//  - null/bool/number/string/array/object map naturally to CBOR
//    nil/bool/float-or-int/text/array/map.
//  - Wrapper objects are recognized and mapped to semantic tags:
//      {"$rfc3339": string}         -> tag(0) RFC3339 time string
//      {"$epoch": number}           -> tag(1) epoch seconds (int or float)
//      {"$decimal":[exp, mant]}     -> tag(4)
//      {"$bigfloat":[exp, mant]}    -> tag(5)
//      {"$base64url": string}       -> tag(21) bytes (base64url)
//      {"$base64": string}          -> tag(22) bytes (base64 std)
//      {"$base16": string}          -> tag(23) bytes (hex)
//      {"$cbor": string}            -> tag(24) embedded CBOR (base64)
//      {"$uri": string}             -> tag(32) URI (text)
//      {"$base64urlstr": string}    -> tag(33) text
//      {"$base64str": string}       -> tag(34) text
//      {"$regex": string}           -> tag(35) regex (text)
//      {"$mime": string}            -> tag(36) MIME (text)
//      {"$uuid": string}            -> tag(37) UUID (canonical hyphenated)
//      {"$selfdescribe": true}      -> tag(55799)
//      {"$tag":N, "$":value}       -> generic tag N
func FromJSONBytes(js []byte) ([]byte, error) {
	dec := json.NewDecoder(strings.NewReader(string(js)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return jsonToCBOR(nil, v)
}

func jsonToCBOR(b []byte, v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return AppendNil(b), nil
	case bool:
		return AppendBool(b, x), nil
	case json.Number:
		// Prefer integers when possible, otherwise float64.
		if strings.ContainsAny(string(x), ".eE") {
			f, err := x.Float64()
			if err != nil {
				return b, err
			}
			return AppendFloat64(b, f), nil
		}
		if i, err := x.Int64(); err == nil {
			return AppendInt64(b, i), nil
		}
		f, err := x.Float64()
		if err != nil {
			return b, err
		}
		return AppendFloat64(b, f), nil
	case float64:
		return AppendFloat64(b, x), nil
	case string:
		return AppendString(b, x), nil
	case []any:
		b = AppendArrayHeader(b, uint32(len(x)))
		var err error
		for _, e := range x {
			b, err = jsonToCBOR(b, e)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	case map[string]any:
		// Wrapper detection
		if out, ok, err := tryWrapper(b, x); ok || err != nil {
			return out, err
		}
		// Plain object
		b = AppendMapHeader(b, uint32(len(x)))
		var err error
		for k, vv := range x {
			b = AppendString(b, k)
			b, err = jsonToCBOR(b, vv)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	default:
		return b, &ErrUnsupportedType{}
	}
}

func tryWrapper(b []byte, m map[string]any) ([]byte, bool, error) {
	// Generic {"$tag":N, "$":value}
	if tagv, ok := m["$tag"]; ok {
		iv, ok2 := m["$"]
		if !ok2 {
			return b, true, errors.New("cbor: $tag wrapper missing $ field")
		}
		tag, err := numToUint64(tagv)
		if err != nil {
			return b, true, err
		}
		inner, err := jsonToCBOR(nil, iv)
		if err != nil {
			return b, true, err
		}
		return AppendTagged(b, tag, inner), true, nil
	}
	// $rfc3339
	if v, ok := m["$rfc3339"]; ok {
		s, _ := v.(string)
		if s == "" {
			return b, true, errors.New("cbor: $rfc3339 expects string")
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return b, true, err
		}
		return AppendRFC3339Time(b, t), true, nil
	}
	// $epoch
	if v, ok := m["$epoch"]; ok {
		// Accept json.Number, float64, or integer-like
		var f float64
		switch vv := v.(type) {
		case json.Number:
			ff, err := vv.Float64()
			if err != nil {
				return b, true, err
			}
			f = ff
		case float64:
			f = vv
		case int64:
			f = float64(vv)
		default:
			return b, true, errors.New("cbor: $epoch expects number")
		}
		sec := mathFloor(f)
		ns := int64(mathRound((f - float64(sec)) * 1e9))
		secs := int64(sec)
		if ns >= 1e9 {
			secs++
			ns -= 1e9
		}
		t := time.Unix(secs, ns).UTC()
		return AppendTime(b, t), true, nil
	}
	// $decimal
	if v, ok := m["$decimal"]; ok {
		arr, ok := v.([]any)
		if !ok || len(arr) != 2 {
			return b, true, errors.New("cbor: $decimal expects [exp, mant]")
		}
		exp, err := anyToInt64(arr[0])
		if err != nil {
			return b, true, err
		}
		mantStr, ok := arr[1].(string)
		if !ok {
			return b, true, errors.New("cbor: $decimal mantissa must be string")
		}
		z, ok := new(big.Int).SetString(mantStr, 10)
		if !ok {
			return b, true, errors.New("cbor: invalid $decimal mantissa")
		}
		return AppendDecimalFraction(b, exp, z), true, nil
	}
	// $bigfloat
	if v, ok := m["$bigfloat"]; ok {
		arr, ok := v.([]any)
		if !ok || len(arr) != 2 {
			return b, true, errors.New("cbor: $bigfloat expects [exp, mant]")
		}
		exp, err := anyToInt64(arr[0])
		if err != nil {
			return b, true, err
		}
		mantStr, ok := arr[1].(string)
		if !ok {
			return b, true, errors.New("cbor: $bigfloat mantissa must be string")
		}
		z, ok := new(big.Int).SetString(mantStr, 10)
		if !ok {
			return b, true, errors.New("cbor: invalid $bigfloat mantissa")
		}
		return AppendBigfloat(b, exp, z), true, nil
	}
	// $base64url -> tagBase64URL
	if v, ok := m["$base64url"]; ok {
		s, _ := v.(string)
		bs, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return b, true, err
		}
		return AppendBase64URL(b, bs), true, nil
	}
	// $base64 -> tagBase64
	if v, ok := m["$base64"]; ok {
		s, _ := v.(string)
		bs, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return b, true, err
		}
		return AppendBase64(b, bs), true, nil
	}
	// $base16 -> tagBase16
	if v, ok := m["$base16"]; ok {
		s, _ := v.(string)
		bs, err := hex.DecodeString(s)
		if err != nil {
			return b, true, err
		}
		return AppendBase16(b, bs), true, nil
	}
	// $cbor -> embedded CBOR (tag24)
	if v, ok := m["$cbor"]; ok {
		s, _ := v.(string)
		bs, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return b, true, err
		}
		return AppendEmbeddedCBOR(b, bs), true, nil
	}
	// $uri
	if v, ok := m["$uri"]; ok {
		s, _ := v.(string)
		if s == "" {
			return b, true, errors.New("cbor: $uri expects string")
		}
		return AppendURI(b, s), true, nil
	}
	// $base64urlstr
	if v, ok := m["$base64urlstr"]; ok {
		s, _ := v.(string)
		return AppendBase64URLString(b, s), true, nil
	}
	// $base64str
	if v, ok := m["$base64str"]; ok {
		s, _ := v.(string)
		return AppendBase64String(b, s), true, nil
	}
	// $regex
	if v, ok := m["$regex"]; ok {
		s, _ := v.(string)
		if s == "" {
			return b, true, errors.New("cbor: $regex expects string")
		}
		if _, err := regexp.Compile(s); err != nil {
			return b, true, err
		}
		return AppendRegexpString(b, s), true, nil
	}
	// $mime
	if v, ok := m["$mime"]; ok {
		s, _ := v.(string)
		if s == "" {
			return b, true, errors.New("cbor: $mime expects string")
		}
		return AppendMIMEString(b, s), true, nil
	}
	// $uuid
	if v, ok := m["$uuid"]; ok {
		s, _ := v.(string)
		if len(s) != 36 { // 8-4-4-4-12
			return b, true, errors.New("cbor: $uuid expects canonical string")
		}
		hexStr := strings.ReplaceAll(s, "-", "")
		bs, err := hex.DecodeString(hexStr)
		if err != nil || len(bs) != 16 {
			return b, true, errors.New("cbor: invalid $uuid")
		}
		var u [16]byte
		copy(u[:], bs)
		return AppendUUID(b, u), true, nil
	}
	// $selfdescribe
	if v, ok := m["$selfdescribe"]; ok {
		bval, _ := v.(bool)
		if !bval {
			return b, true, errors.New("cbor: $selfdescribe expects true")
		}
		return AppendSelfDescribeCBOR(b), true, nil
	}
	return b, false, nil
}

func numToUint64(v any) (uint64, error) {
	switch t := v.(type) {
	case json.Number:
		if strings.ContainsAny(string(t), ".eE") {
			f, err := t.Float64()
			if err != nil {
				return 0, err
			}
			if f < 0 {
				return 0, errors.New("cbor: negative tag")
			}
			return uint64(f), nil
		}
		i, err := t.Int64()
		if err != nil {
			return 0, err
		}
		if i < 0 {
			return 0, errors.New("cbor: negative tag")
		}
		return uint64(i), nil
	case float64:
		if t < 0 {
			return 0, errors.New("cbor: negative tag")
		}
		return uint64(t), nil
	case int64:
		if t < 0 {
			return 0, errors.New("cbor: negative tag")
		}
		return uint64(t), nil
	case int:
		if t < 0 {
			return 0, errors.New("cbor: negative tag")
		}
		return uint64(t), nil
	default:
		return 0, errors.New("cbor: expected numeric tag")
	}
}

func anyToInt64(v any) (int64, error) {
	switch t := v.(type) {
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return 0, err
		}
		return i, nil
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	default:
		return 0, errors.New("cbor: expected integer")
	}
}

func mathFloor(f float64) float64 {
	if f >= 0 {
		return float64(int64(f))
	}
	if float64(int64(f)) == f {
		return f
	}
	return float64(int64(f) - 1)
}

func mathRound(f float64) float64 {
	if f >= 0 {
		return float64(int64(f + 0.5))
	}
	return float64(int64(f - 0.5))
}
