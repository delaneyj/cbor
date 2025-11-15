package cbor

import (
    "encoding/hex"
    "fmt"
    "math"
    "strconv"
)

// DiagBytes renders the next CBOR item in RFC diagnostic notation and returns the remaining bytes.
func DiagBytes(b []byte) (string, []byte, error) {
    bb := GetByteBuffer()
    defer PutByteBuffer(bb)
    rest, err := diagOneBuf(bb, b, 0)
    if err != nil {
        return "", b, err
    }
    out := make([]byte, bb.Len())
    copy(out, bb.Bytes())
    return string(out), rest, nil
}

func diagOneBuf(buf *ByteBuffer, b []byte, depth int) ([]byte, error) {
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
        if err != nil { return b, err }
        buf.WriteString(strconv.FormatUint(u, 10))
        return o, nil
    case majorTypeNegInt:
        u, o, err := readUintCore(b, majorTypeNegInt)
        if err != nil { return b, err }
        n := int64(-1) - int64(u)
        buf.WriteString(strconv.FormatInt(n, 10))
        return o, nil
    case majorTypeBytes:
        if add == addInfoIndefinite {
            p := b[1:]
            buf.WriteString("(_")
            first := true
            for {
                if len(p) < 1 { return b, ErrShortBytes }
                if p[0] == makeByte(majorTypeSimple, simpleBreak) {
                    buf.WriteString(")")
                    return p[1:], nil
                }
                sz, o, err := readUintCore(p, majorTypeBytes)
                if err != nil { return b, err }
                if uint64(len(o)) < sz { return b, ErrShortBytes }
                if !first { buf.WriteString(", ") } else { buf.WriteString(" "); first = false }
                buf.WriteString("h'")
                d := buf.Extend(hex.EncodedLen(int(sz)))
                hex.Encode(d, o[:sz])
                buf.WriteString("'")
                p = o[sz:]
            }
        }
        bs, o, err := ReadBytesBytes(b, nil)
        if err != nil { return b, err }
        buf.WriteString("h'")
        d := buf.Extend(hex.EncodedLen(len(bs)))
        hex.Encode(d, bs)
        buf.WriteString("'")
        return o, nil
    case majorTypeText:
        if add == addInfoIndefinite {
            p := b[1:]
            buf.WriteString("(_")
            first := true
            for {
                if len(p) < 1 { return b, ErrShortBytes }
                if p[0] == makeByte(majorTypeSimple, simpleBreak) {
                    buf.WriteString(")")
                    return p[1:], nil
                }
                chunk, o, err := ReadStringZC(p)
                if err != nil { return b, err }
                q := strconv.Quote(string(chunk))
                if !first { buf.WriteString(", ") } else { buf.WriteString(" "); first = false }
                buf.WriteString(q)
                p = o
            }
        }
        s, o, err := ReadStringBytes(b)
        if err != nil { return b, err }
        buf.WriteString(strconv.Quote(s))
        return o, nil
    case majorTypeArray:
        if add == addInfoIndefinite {
            p := b[1:]
            buf.WriteString("[_")
            first := true
            for {
                if len(p) < 1 { return b, ErrShortBytes }
                if p[0] == makeByte(majorTypeSimple, simpleBreak) {
                    buf.WriteString("]")
                    return p[1:], nil
                }
                if !first { buf.WriteString(", ") } else { buf.WriteString(" "); first = false }
                var err error
                p, err = diagOneBuf(buf, p, depth+1)
                if err != nil { return b, err }
            }
        }
        sz, p, err := readUintCore(b, majorTypeArray)
        if err != nil { return b, err }
        buf.WriteString("[")
        for i := uint64(0); i < sz; i++ {
            if i > 0 { buf.WriteString(", ") }
            var err error
            p, err = diagOneBuf(buf, p, depth+1)
            if err != nil { return b, err }
        }
        buf.WriteString("]")
        return p, nil
    case majorTypeMap:
        if add == addInfoIndefinite {
            p := b[1:]
            buf.WriteString("{_")
            first := true
            for {
                if len(p) < 1 { return b, ErrShortBytes }
                if p[0] == makeByte(majorTypeSimple, simpleBreak) {
                    buf.WriteString("}")
                    return p[1:], nil
                }
                if !first { buf.WriteString(", ") } else { buf.WriteString(" "); first = false }
                // key
                var err error
                p, err = diagOneBuf(buf, p, depth+1)
                if err != nil { return b, err }
                buf.WriteString(": ")
                // value
                p, err = diagOneBuf(buf, p, depth+1)
                if err != nil { return b, err }
            }
        }
        sz, p, err := readUintCore(b, majorTypeMap)
        if err != nil { return b, err }
        buf.WriteString("{")
        for i := uint64(0); i < sz; i++ {
            if i > 0 { buf.WriteString(", ") }
            var err error
            p, err = diagOneBuf(buf, p, depth+1) // key
            if err != nil { return b, err }
            buf.WriteString(": ")
            p, err = diagOneBuf(buf, p, depth+1) // value
            if err != nil { return b, err }
        }
        buf.WriteString("}")
        return p, nil
    case majorTypeTag:
        tag, o, err := ReadTagBytes(b)
        if err != nil { return b, err }
        buf.WriteString(strconv.FormatUint(tag, 10))
        buf.WriteString("(")
        o2, err := diagOneBuf(buf, o, depth+1)
        if err != nil { return b, err }
        buf.WriteString(")")
        return o2, nil
    case majorTypeSimple:
        switch add {
        case simpleFalse:
            buf.WriteString("false")
            return b[1:], nil
        case simpleTrue:
            buf.WriteString("true")
            return b[1:], nil
        case simpleNull:
            buf.WriteString("null")
            return b[1:], nil
        case simpleUndefined:
            buf.WriteString("undefined")
            return b[1:], nil
        case simpleFloat16:
            f, o, err := ReadFloat16Bytes(b)
            if err != nil { return b, err }
            buf.WriteString(formatFloat32Diag(f))
            return o, nil
        case simpleFloat32:
            f, o, err := ReadFloat32Bytes(b)
            if err != nil { return b, err }
            buf.WriteString(formatFloat32Diag(f))
            return o, nil
        case simpleFloat64:
            f, o, err := ReadFloat64Bytes(b)
            if err != nil { return b, err }
            buf.WriteString(formatFloat64Diag(f))
            return o, nil
        default:
            if add < 20 {
                buf.WriteString(fmt.Sprintf("simple(%d)", add))
                return b[1:], nil
            }
            if add == addInfoUint8 {
                if len(b) < 2 { return b, ErrShortBytes }
                buf.WriteString(fmt.Sprintf("simple(%d)", b[1]))
                return b[2:], nil
            }
            return b, &ErrUnsupportedType{}
        }
    }
    return b, &ErrUnsupportedType{}
}

// formatFloat64Diag returns a diagnostic string for float64 matching RFC examples
func formatFloat64Diag(f float64) string {
    if math.IsInf(f, +1) { return "Infinity" }
    if math.IsInf(f, -1) { return "-Infinity" }
    if math.IsNaN(f) { return "NaN" }
    af := math.Abs(f)
    // Prefer fixed-point for reasonable magnitudes
    if af == 0 || af < 1e15 {
        s := strconv.FormatFloat(f, 'f', -1, 64)
        return trimTrailingZerosDot(s)
    }
    return strconv.FormatFloat(f, 'g', -1, 64)
}

// formatFloat32Diag returns a diagnostic string for float32 matching RFC examples
func formatFloat32Diag(f float32) string {
    if math.IsInf(float64(f), +1) { return "Infinity" }
    if math.IsInf(float64(f), -1) { return "-Infinity" }
    if math.IsNaN(float64(f)) { return "NaN" }
    af := math.Abs(float64(f))
    if af == 0 || af < 1e15 {
        s := strconv.FormatFloat(float64(f), 'f', -1, 32)
        return trimTrailingZerosDot(s)
    }
    return strconv.FormatFloat(float64(f), 'g', -1, 32)
}

func trimTrailingZerosDot(s string) string {
    // Trim trailing zeros and optional dot
    i := len(s)
    for i > 0 && s[i-1] == '0' { i-- }
    if i > 0 && s[i-1] == '.' { i-- }
    return s[:i]
}
