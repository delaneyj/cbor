package cbor

import "unsafe"

// UnsafeString returns a string that shares the same underlying
// memory as b. It must only be used in Trusted decode paths where
// the backing buffer is immutable for the lifetime of the string.
func UnsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// UnsafeBytes returns the string as a byte slice. It is
// equivalent to []byte(s) and retained for compatibility.
func UnsafeBytes(s string) []byte { return []byte(s) }
