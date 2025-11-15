package cbor

// Worst-case encoded sizes for common types. For variable-length types
// such as strings and byte slices, the total encoded size is the
// corresponding prefix size plus the length of the value.
const (
	Int64Size      = 9
	IntSize        = Int64Size
	UintSize       = Int64Size
	Int8Size       = 2
	Int16Size      = 3
	Int32Size      = 5
	Uint8Size      = 2
	Uint16Size     = 3
	Uint32Size     = 5
	Uint64Size     = Int64Size
	Float64Size    = 9
	Float32Size    = 5
	DurationSize   = Int64Size
	TimeSize       = 15
	BoolSize       = 1
	NilSize        = 1
	MapHeaderSize   = 5
	ArrayHeaderSize = 5
	BytesPrefixSize     = 5
	StringPrefixSize    = 5
	ExtensionPrefixSize = 6
)

