package cbor

import "unicode/utf8"

// isUTF8Valid validates UTF-8 for a byte slice. It can be overridden by
// architecture-specific, SIMD-accelerated implementations via build tags.
var isUTF8Valid = func(b []byte) bool { return utf8.Valid(b) }

