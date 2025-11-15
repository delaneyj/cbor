package structs

// Containers exercises slices and maps of struct and pointer-to-struct
// fields to validate generated DecodeSafe/DecodeTrusted for container
// element types.
type Containers struct {
	Items   []Scalars         `cbor:"items"`
	Ptrs    []*Scalars        `cbor:"ptrs"`
	Map     map[string]Scalars  `cbor:"map"`
	PtrMap  map[string]*Scalars `cbor:"ptr_map"`
}

