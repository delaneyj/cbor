package cbor

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

// ToJSONBytes converts the next CBOR item into a JSON encoding and
// returns the JSON bytes and remainder. It follows a similar mapping
// as FromJSONBytes, including unwrapping common semantic tags into
// wrapper objects where appropriate.
func ToJSONBytes(b []byte) ([]byte, []byte, error) {
	bb := GetByteBuffer()
	defer PutByteBuffer(bb)
	rest, err := toJSON(bb, b, 0)
	if err != nil {
		return nil, b, err
	}
	out := make([]byte, bb.Len())
	copy(out, bb.Bytes())
	return out, rest, nil
}

func toJSON(buf *ByteBuffer, b []byte, depth int) ([]byte, error) {
	if depth > recursionLimit {
		return b, ErrMaxDepthExceeded
	}
	if len(b) < 1 {
		return b, ErrShortBytes
	}
	maj := getMajorType(b[0])
	add := getAddInfo(b[0])

	switch maj {
	case majorTypeUint:
		u, o, err := readUintCore(b, majorTypeUint)
		if err != nil {
			return b, err
		}
		buf.WriteString(strconv.FormatUint(u, 10))
		return o, nil
	case majorTypeNegInt:
		u, o, err := readUintCore(b, majorTypeNegInt)
		if err != nil {
			return b, err
		}
		n := int64(-1) - int64(u)
		buf.WriteString(strconv.FormatInt(n, 10))
		return o, nil
	case majorTypeBytes:
		bs, o, err := ReadBytesBytes(b, nil)
		if err != nil {
			return b, err
		}
		// base64-encode byte strings
		buf.WriteString("\"")
		encodeBase64Std(buf, bs)
		buf.WriteString("\"")
		return o, nil
	case majorTypeText:
		s, o, err := ReadStringBytes(b)
		if err != nil {
			return b, err
		}
		js, _ := json.Marshal(s)
		buf.Write(js)
		return o, nil
	case majorTypeArray:
		if add == addInfoIndefinite {
			buf.WriteString("[")
			p := b[1:]
			first := true
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					buf.WriteString("]")
					return p[1:], nil
				}
				if !first {
					buf.WriteString(",")
				} else {
					first = false
				}
				var err error
				p, err = toJSON(buf, p, depth+1)
				if err != nil {
					return b, err
				}
			}
		}
		sz, p, err := readUintCore(b, majorTypeArray)
		if err != nil {
			return b, err
		}
		buf.WriteString("[")
		for i := uint64(0); i < sz; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			var err error
			p, err = toJSON(buf, p, depth+1)
			if err != nil {
				return b, err
			}
		}
		buf.WriteString("]")
		return p, nil
	case majorTypeMap:
		if add == addInfoIndefinite {
			buf.WriteString("{")
			p := b[1:]
			first := true
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					buf.WriteString("}")
					return p[1:], nil
				}
				// key must be JSON string; try fast path
				if getMajorType(p[0]) == majorTypeText {
					k, o, err := ReadStringBytes(p)
					if err != nil {
						return b, err
					}
					if !first {
						buf.WriteString(",")
					} else {
						first = false
					}
					kj, _ := json.Marshal(k)
					buf.Write(kj)
					buf.WriteString(":")
					var err2 error
					p, err2 = toJSON(buf, o, depth+1)
					if err2 != nil {
						return b, err2
					}
				} else {
					// fallback: use diagnostic notation as key
					ks, o, err := DiagBytes(p)
					if err != nil {
						return b, err
					}
					if !first {
						buf.WriteString(",")
					} else {
						first = false
					}
					kj, _ := json.Marshal(ks)
					buf.Write(kj)
					buf.WriteString(":")
					var err2 error
					p, err2 = toJSON(buf, o, depth+1)
					if err2 != nil {
						return b, err2
					}
				}
			}
		}
		sz, p, err := readUintCore(b, majorTypeMap)
		if err != nil {
			return b, err
		}
		buf.WriteString("{")
		for i := uint64(0); i < sz; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			if getMajorType(p[0]) == majorTypeText {
				k, o, err := ReadStringBytes(p)
				if err != nil {
					return b, err
				}
				kj, _ := json.Marshal(k)
				buf.Write(kj)
				buf.WriteString(":")
				var err2 error
				p, err2 = toJSON(buf, o, depth+1)
				if err2 != nil {
					return b, err2
				}
			} else {
				ks, o, err := DiagBytes(p)
				if err != nil {
					return b, err
				}
				kj, _ := json.Marshal(ks)
				buf.Write(kj)
				buf.WriteString(":")
				var err2 error
				p, err2 = toJSON(buf, o, depth+1)
				if err2 != nil {
					return b, err2
				}
			}
		}
		buf.WriteString("}")
		return p, nil
	case majorTypeTag:
		tag, o, err := ReadTagBytes(b)
		if err != nil {
			return b, err
		}
		switch tag {
		case tagDateTimeString: // 0
			tm, rest, err := ReadRFC3339TimeBytes(b)
			if err != nil {
				return b, err
			}
			js, _ := json.Marshal(tm.Format(time.RFC3339Nano))
			buf.Write(js)
			return rest, nil
		case tagEpochDateTime: // 1
			tm, rest, err := ReadTimeBytes(b)
			if err != nil {
				return b, err
			}
			js, _ := json.Marshal(tm.Format(time.RFC3339Nano))
			buf.Write(js)
			return rest, nil
		case tagPosBignum, tagNegBignum: // 2/3
			z, rest, err := ReadBigIntBytes(b)
			if err != nil {
				return b, err
			}
			js, _ := json.Marshal(z.String())
			buf.Write(js)
			return rest, nil
		case tagDecimalFrac: // 4 -> {"$decimal":[exp, mant]}
			exp, mant, rest, err := ReadDecimalFractionBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$decimal":[`)
			buf.WriteString(strconv.FormatInt(exp, 10))
			buf.WriteString(",")
			ms, _ := json.Marshal(mant.String())
			buf.Write(ms)
			buf.WriteString("]}")
			return rest, nil
		case tagBigfloat: // 5 -> {"$bigfloat":[exp, mant]}
			exp, mant, rest, err := ReadBigfloatBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$bigfloat":[`)
			buf.WriteString(strconv.FormatInt(exp, 10))
			buf.WriteString(",")
			ms, _ := json.Marshal(mant.String())
			buf.Write(ms)
			buf.WriteString("]}")
			return rest, nil
		case tagBase64URL: // 21 -> {"$base64url":"..."}
			bs, rest, err := ReadBase64URLBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$base64url":"`)
			encodeBase64RawURL(buf, bs)
			buf.WriteString(`"}`)
			return rest, nil
		case tagBase64: // 22 -> {"$base64":"..."}
			bs, rest, err := ReadBase64Bytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$base64":"`)
			encodeBase64Std(buf, bs)
			buf.WriteString(`"}`)
			return rest, nil
		case tagBase16: // 23 -> {"$base16":"..."}
			bs, rest, err := ReadBase16Bytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$base16":"`)
			d := buf.Extend(hex.EncodedLen(len(bs)))
			hex.Encode(d, bs)
			buf.WriteString(`"}`)
			return rest, nil
		case tagCBOR: // 24 embedded CBOR -> {"$cbor":"..."}
			payload, rest, err := ReadEmbeddedCBORBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$cbor":"`)
			encodeBase64Std(buf, payload)
			buf.WriteString(`"}`)
			return rest, nil
		case tagURI: // 32 -> plain JSON string
			s, rest, err := ReadURIStringBytes(b)
			if err != nil {
				return b, err
			}
			js, _ := json.Marshal(s)
			buf.Write(js)
			return rest, nil
		case tagBase64URLString: // 33 -> {"$base64urlstr":string}
			s, rest, err := ReadBase64URLStringBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$base64urlstr":`)
			js, _ := json.Marshal(s)
			buf.Write(js)
			buf.WriteString("}")
			return rest, nil
		case tagBase64String: // 34 -> {"$base64str":string}
			s, rest, err := ReadBase64StringBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$base64str":`)
			js, _ := json.Marshal(s)
			buf.Write(js)
			buf.WriteString("}")
			return rest, nil
		case tagRegexp: // 35 -> {"$regex":string}
			s, rest, err := ReadRegexpStringBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$regex":`)
			js, _ := json.Marshal(s)
			buf.Write(js)
			buf.WriteString("}")
			return rest, nil
		case tagMIME: // 36 -> {"$mime":string}
			s, rest, err := ReadMIMEStringBytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(`{"$mime":`)
			js, _ := json.Marshal(s)
			buf.Write(js)
			buf.WriteString("}")
			return rest, nil
		case 37: // UUID -> {"$uuid":"..."}
			u, rest, err := ReadUUIDBytes(b)
			if err != nil {
				return b, err
			}
			hexs := hex.EncodeToString(u[:])
			uuidStr := hexs[0:8] + "-" + hexs[8:12] + "-" + hexs[12:16] + "-" + hexs[16:20] + "-" + hexs[20:32]
			buf.WriteString(`{"$uuid":"`)
			buf.WriteString(uuidStr)
			buf.WriteString(`"}`)
			return rest, nil
		case tagSelfDescribeCBOR: // 55799 -> {"$selfdescribe":true}
			_, found, _ := StripSelfDescribeCBOR(b)
			if found {
				buf.WriteString(`{"$selfdescribe":true}`)
				_, o2, _ := ReadTagBytes(b)
				return o2, nil
			}
			return b, &ErrUnsupportedType{}
		default:
			// Generic: {"$tag":N, "$": value}
			vbuf := GetByteBuffer()
			rest, err := toJSON(vbuf, o, depth+1)
			if err != nil {
				PutByteBuffer(vbuf)
				return b, err
			}
			buf.WriteString(`{"$tag":`)
			buf.WriteString(strconv.FormatUint(tag, 10))
			buf.WriteString(`,"$":`)
			buf.Write(vbuf.Bytes())
			buf.WriteString("}")
			PutByteBuffer(vbuf)
			return rest, nil
		}
	case majorTypeSimple:
		switch add {
		case simpleFalse:
			buf.WriteString("false")
			return b[1:], nil
		case simpleTrue:
			buf.WriteString("true")
			return b[1:], nil
		case simpleNull, simpleUndefined:
			buf.WriteString("null")
			return b[1:], nil
		case simpleFloat16:
			f, o, err := ReadFloat16Bytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
			return o, nil
		case simpleFloat32:
			f, o, err := ReadFloat32Bytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(strconv.FormatFloat(float64(f), 'g', -1, 32))
			return o, nil
		case simpleFloat64:
			f, o, err := ReadFloat64Bytes(b)
			if err != nil {
				return b, err
			}
			buf.WriteString(strconv.FormatFloat(f, 'g', -1, 64))
			return o, nil
		case addInfoUint8:
			// application-defined simple value; map to null in JSON by default
			if len(b) < 2 {
				return b, ErrShortBytes
			}
			buf.WriteString("null")
			return b[2:], nil
		default:
			if add < 20 {
				// unassigned simple values -> null
				buf.WriteString("null")
				return b[1:], nil
			}
			return b, &ErrUnsupportedType{}
		}
	}
	return b, &ErrUnsupportedType{}
}

// encodeBase64Std writes standard base64 of src into buf.
func encodeBase64Std(buf *ByteBuffer, src []byte) {
	out := buf.Extend(base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(out, src)
}

// encodeBase64RawURL writes raw base64url of src into buf.
func encodeBase64RawURL(buf *ByteBuffer, src []byte) {
	out := buf.Extend(base64.RawURLEncoding.EncodedLen(len(src)))
	base64.RawURLEncoding.Encode(out, src)
}

