package cbor

import ()

// ValidateWellFormedBytes validates that the next CBOR data item in b is well-formed per RFC 8949
// and returns the remaining bytes after that item.
// Checks performed:
// - Structural correctness of arrays, maps, tags, simple values
// - String UTF-8 validity (for major type 3)
// - Prohibits reserved additional info values 28,29,30
func ValidateWellFormedBytes(b []byte) (rest []byte, err error) {
	return validateWellFormed(b, 0)
}

// ValidateDocument validates that all items in b are well-formed until input is exhausted.
func ValidateDocument(b []byte) error {
	var err error
	for len(b) > 0 {
		b, err = validateWellFormed(b, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateWellFormed(b []byte, depth int) ([]byte, error) {
	if depth > recursionLimit {
		return b, ErrMaxDepthExceeded
	}
	if len(b) < 1 {
		return b, ErrShortBytes
	}
	lead := b[0]
	major := getMajorType(lead)
	add := getAddInfo(lead)

	// Reserved additional info values 28, 29, 30 are not well-formed
	if add == 28 || add == 29 || add == 30 {
		return b, InvalidPrefixError{Want: major, Got: major}
	}

	switch major {
	case majorTypeUint, majorTypeNegInt, majorTypeTag:
		_, o, err := readUintCore(b, major)
		if err != nil {
			return b, err
		}
		if major == majorTypeTag {
			return validateWellFormed(o, depth+1)
		}
		return o, nil

	case majorTypeBytes:
		if add == addInfoIndefinite {
			// indefinite bytes: series of definite byte strings terminated by break
			p := b[1:]
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					return p[1:], nil
				}
				// chunk must be bytes
				sz, o, err := readUintCore(p, majorTypeBytes)
				if err != nil {
					return b, err
				}
				if uint64(len(o)) < sz {
					return b, ErrShortBytes
				}
				p = o[sz:]
			}
		}
		sz, o, err := readUintCore(b, majorTypeBytes)
		if err != nil {
			return b, err
		}
		if uint64(len(o)) < sz {
			return b, ErrShortBytes
		}
		return o[sz:], nil

	case majorTypeText:
		if add == addInfoIndefinite {
			p := b[1:]
			// accumulate chunks and validate utf-8 progressively
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					return p[1:], nil
				}
				// chunk must be text
				chunk, o, err := ReadStringZC(p)
				if err != nil {
					return b, err
				}
            if !isUTF8Valid(chunk) {
                return b, ErrInvalidUTF8
            }
				p = o
			}
		}
		// definite string
		s, o, err := ReadStringZC(b)
		if err != nil {
			return b, err
		}
        if !isUTF8Valid(s) {
            return b, ErrInvalidUTF8
        }
		return o, nil

	case majorTypeArray:
		if add == addInfoIndefinite {
			p := b[1:]
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					return p[1:], nil
				}
				var err error
				p, err = validateWellFormed(p, depth+1)
				if err != nil {
					return b, err
				}
			}
		}
		sz, p, err := readUintCore(b, majorTypeArray)
		if err != nil {
			return b, err
		}
		for i := uint64(0); i < sz; i++ {
			p, err = validateWellFormed(p, depth+1)
			if err != nil {
				return b, err
			}
		}
		return p, nil

	case majorTypeMap:
		if add == addInfoIndefinite {
			p := b[1:]
			for {
				if len(p) < 1 {
					return b, ErrShortBytes
				}
				if p[0] == makeByte(majorTypeSimple, simpleBreak) {
					return p[1:], nil
				}
				var err error
				p, err = validateWellFormed(p, depth+1) // key
				if err != nil {
					return b, err
				}
				p, err = validateWellFormed(p, depth+1) // value
				if err != nil {
					return b, err
				}
			}
		}
		sz, p, err := readUintCore(b, majorTypeMap)
		if err != nil {
			return b, err
		}
		for i := uint64(0); i < sz; i++ {
			p, err = validateWellFormed(p, depth+1) // key
			if err != nil {
				return b, err
			}
			p, err = validateWellFormed(p, depth+1) // value
			if err != nil {
				return b, err
			}
		}
		return p, nil

	case majorTypeSimple:
		switch add {
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
		case addInfoUint8: // one-byte simple value (0xf8 xx)
			if len(b) < 2 {
				return b, ErrShortBytes
			}
			return b[2:], nil
		default:
			if add < 20 { // unassigned simple values are still well-formed
				return b[1:], nil
			}
			return b, &ErrUnsupportedType{}
		}
	}
	return b, &ErrUnsupportedType{}
}
