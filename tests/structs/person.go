package structs

// Person is a simple example type used to exercise
// struct code generation semantics (map encoding,
// omitempty, and Safe vs Trusted decode paths).
type Person struct {
	Name string `cbor:"name"`
	Age  int    `cbor:"age,omitempty"`
	Data []byte `cbor:"data"`
}

