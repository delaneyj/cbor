package cbor

// Writer provides a minimal CBOR writer backed by ByteBuffer.
// It is intended for use by generated EncodeMsg implementations.
type Writer struct {
	bb *ByteBuffer
}

// NewWriter constructs a Writer that appends to the provided ByteBuffer.
func NewWriter(bb *ByteBuffer) *Writer { return &Writer{bb: bb} }

// Bytes returns the underlying encoded bytes.
func (w *Writer) Bytes() []byte { return w.bb.Bytes() }

// WriteMapHeader writes a map header with the given size.
func (w *Writer) WriteMapHeader(sz uint32) error {
	w.bb.AppendMapHeader(sz)
	return nil
}

// WriteString writes a text string value.
func (w *Writer) WriteString(s string) error {
	w.bb.AppendString(s)
	return nil
}

// WriteBool writes a bool value.
func (w *Writer) WriteBool(v bool) error {
	w.bb.AppendBool(v)
	return nil
}

// WriteInt writes an int value.
func (w *Writer) WriteInt(v int) error {
	w.bb.AppendInt64(int64(v))
	return nil
}

// WriteInt64 writes an int64 value.
func (w *Writer) WriteInt64(v int64) error {
	w.bb.AppendInt64(v)
	return nil
}

// WriteUint writes a uint value.
func (w *Writer) WriteUint(v uint) error {
	w.bb.AppendUint64(uint64(v))
	return nil
}

// WriteUint64 writes a uint64 value.
func (w *Writer) WriteUint64(v uint64) error {
	w.bb.AppendUint64(v)
	return nil
}

// WriteFloat32 writes a float32 value.
func (w *Writer) WriteFloat32(v float32) error {
	w.bb.AppendFloat32(v)
	return nil
}

// WriteFloat64 writes a float64 value.
func (w *Writer) WriteFloat64(v float64) error {
	w.bb.AppendFloat64(v)
	return nil
}

// WriteBytes writes a byte string value.
func (w *Writer) WriteBytes(v []byte) error {
	w.bb.AppendBytes(v)
	return nil
}
