package structs

import "time"

// Scalars exercises a wide range of primitive field types to
// validate struct code generation for DecodeSafe/DecodeTrusted.
type Scalars struct {
	S  string  `cbor:"s"`
	B  bool    `cbor:"b"`
	I  int     `cbor:"i"`
	I8 int8    `cbor:"i8"`
	I16 int16  `cbor:"i16"`
	I32 int32  `cbor:"i32"`
	I64 int64  `cbor:"i64"`
	U  uint    `cbor:"u"`
	U8 uint8   `cbor:"u8"`
	U16 uint16 `cbor:"u16"`
	U32 uint32 `cbor:"u32"`
	U64 uint64 `cbor:"u64"`
	F32 float32 `cbor:"f32"`
	F64 float64 `cbor:"f64"`
	Data []byte `cbor:"data"`
	Ints []int `cbor:"ints"`
	Names []string `cbor:"names"`
	Scores map[string]int `cbor:"scores"`
	T  time.Time     `cbor:"t"`
	D  time.Duration `cbor:"d"`
}

// Nested exercises nested struct and pointer fields.
type Nested struct {
	ID   string   `cbor:"id"`
	Base Scalars  `cbor:"base"`
	Ptr  *Scalars `cbor:"ptr,omitempty"`
}
