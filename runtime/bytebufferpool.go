package cbor

import (
    "io"
    "sync"
)

// Local byte buffer pool under our control.
//
// Guidelines:
// - Do not call Reset() before Put() unless you intend to reuse the buffer
//   before putting it back. The pool does not require Reset() before Put().
// - Use Ensure(n) to grow capacity up-front when you know you will append
//   at least n more bytes. This avoids repeated reallocations.

type ByteBuffer struct {
    b []byte
}

var bbPool = sync.Pool{New: func() any { return &ByteBuffer{b: make([]byte, 0, 1024)} }}

// GetByteBuffer obtains a pooled ByteBuffer. The buffer is Reset() before
// being returned so length is zero (capacity may be reused).
func GetByteBuffer() *ByteBuffer {
    bb := bbPool.Get().(*ByteBuffer)
    bb.Reset()
    return bb
}

// GetMinSize obtains a pooled ByteBuffer with capacity for at least size bytes.
// The buffer is Reset() and then grown if needed.
func GetMinSize(size int) *ByteBuffer {
    bb := bbPool.Get().(*ByteBuffer)
    bb.Reset()
    if size > 0 { bb.Ensure(size) }
    return bb
}

// PutByteBuffer returns the buffer to the pool. The content is left intact
// (no implicit Reset). Call Reset() yourself if you want to clear before reuse
// without returning to the pool.
// PutByteBuffer returns the buffer to the pool after Resetting length to zero.
func PutByteBuffer(bb *ByteBuffer) { bb.Reset(); bbPool.Put(bb) }

// Bytes returns the underlying bytes.
func (bb *ByteBuffer) Bytes() []byte { return bb.b }

// Len returns length.
func (bb *ByteBuffer) Len() int { return len(bb.b) }

// Cap returns capacity.
func (bb *ByteBuffer) Cap() int { return cap(bb.b) }

// Reset resets the length to zero; capacity is unchanged.
func (bb *ByteBuffer) Reset() { bb.b = bb.b[:0] }

// Ensure ensures there is room for at least n more bytes without reallocation.
// If needed, it grows the underlying slice.
func (bb *ByteBuffer) Ensure(n int) {
    need := len(bb.b) + n
    if cap(bb.b) >= need {
        return
    }
    // Grow: double until enough, then allocate
    c := cap(bb.b)
    if c == 0 {
        c = 1024
    }
    for c < need { c <<= 1 }
    nb := make([]byte, len(bb.b), c)
    copy(nb, bb.b)
    bb.b = nb
}

// Extend grows the buffer by n bytes and returns a slice to the newly
// appended region for direct writes. The buffer length is advanced by n.
func (bb *ByteBuffer) Extend(n int) []byte {
    old := len(bb.b)
    bb.Ensure(n)
    bb.b = bb.b[:old+n]
    return bb.b[old:]
}

// Write implements io.Writer.
func (bb *ByteBuffer) Write(p []byte) (int, error) {
    bb.Ensure(len(p))
    bb.b = append(bb.b, p...)
    return len(p), nil
}

// WriteString appends a string.
func (bb *ByteBuffer) WriteString(s string) (int, error) {
    bb.Ensure(len(s))
    bb.b = append(bb.b, s...)
    return len(s), nil
}

// WriteByte appends a single byte.
func (bb *ByteBuffer) WriteByte(c byte) error {
    bb.Ensure(1)
    bb.b = append(bb.b, c)
    return nil
}

// ReadFrom implements io.ReaderFrom for efficient streaming into the buffer.
func (bb *ByteBuffer) ReadFrom(r io.Reader) (int64, error) {
    var total int64
    for {
        // Grow a chunk (~32KB) if no free space
        if cap(bb.b)-len(bb.b) < 32*1024 { bb.Ensure(32 * 1024) }
        // Read into free tail
        n, err := r.Read(bb.b[len(bb.b):cap(bb.b)])
        if n > 0 {
            bb.b = bb.b[:len(bb.b)+n]
            total += int64(n)
        }
        if err != nil {
            if err == io.EOF { return total, nil }
            return total, err
        }
    }
}

// Convenience CBOR appenders on ByteBuffer
// These mirror the package-level AppendXxx functions but operate directly on
// the ByteBuffer, enabling a fluent, zero-alloc style when combined with
// upfront Ensure()/GetMinSize().

func (bb *ByteBuffer) AppendMapHeader(sz uint32) *ByteBuffer {
    bb.b = AppendMapHeader(bb.b, sz)
    return bb
}

func (bb *ByteBuffer) AppendArrayHeader(sz uint32) *ByteBuffer {
    bb.b = AppendArrayHeader(bb.b, sz)
    return bb
}

func (bb *ByteBuffer) AppendArrayHeaderIndefinite() *ByteBuffer {
    bb.b = AppendArrayHeaderIndefinite(bb.b)
    return bb
}

func (bb *ByteBuffer) AppendBreak() *ByteBuffer {
    bb.b = AppendBreak(bb.b)
    return bb
}

func (bb *ByteBuffer) AppendString(s string) *ByteBuffer {
    bb.b = AppendString(bb.b, s)
    return bb
}

func (bb *ByteBuffer) AppendBytes(bs []byte) *ByteBuffer {
    bb.b = AppendBytes(bb.b, bs)
    return bb
}

func (bb *ByteBuffer) AppendInt64(i int64) *ByteBuffer {
    bb.b = AppendInt64(bb.b, i)
    return bb
}

func (bb *ByteBuffer) AppendUint64(u uint64) *ByteBuffer {
    bb.b = AppendUint64(bb.b, u)
    return bb
}

func (bb *ByteBuffer) AppendBool(v bool) *ByteBuffer {
    bb.b = AppendBool(bb.b, v)
    return bb
}

func (bb *ByteBuffer) AppendFloat64(f float64) *ByteBuffer {
    bb.b = AppendFloat64(bb.b, f)
    return bb
}

func (bb *ByteBuffer) AppendFloat32(f float32) *ByteBuffer {
    bb.b = AppendFloat32(bb.b, f)
    return bb
}

func (bb *ByteBuffer) AppendTag(tag uint64) *ByteBuffer {
    bb.b = AppendTag(bb.b, tag)
    return bb
}
