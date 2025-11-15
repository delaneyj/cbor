package benchmarks

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
	msgp "github.com/tinylib/msgp/msgp"
)

// Primitive encode microbenchmarks comparing this CBOR runtime against
// tinylib/msgp's MessagePack runtime for similar operations. This
// helps surface regressions relative to the original msgp-inspired
// implementation.

func BenchmarkCBOR_AppendInt64(b *testing.B) {
	var out []byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = cbor.AppendInt64(out[:0], int64(i))
	}
	_ = out
}

func BenchmarkMsgp_AppendInt64(b *testing.B) {
	var out []byte
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = msgp.AppendInt64(out[:0], int64(i))
	}
	_ = out
}

func BenchmarkCBOR_AppendString(b *testing.B) {
	var out []byte
	s := "hello world"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = cbor.AppendString(out[:0], s)
	}
	_ = out
}

func BenchmarkMsgp_AppendString(b *testing.B) {
	var out []byte
	s := "hello world"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = msgp.AppendString(out[:0], s)
	}
	_ = out
}

func BenchmarkCBOR_AppendBytes(b *testing.B) {
	var out []byte
	data := []byte("payload bytes")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = cbor.AppendBytes(out[:0], data)
	}
	_ = out
}

func BenchmarkMsgp_AppendBytes(b *testing.B) {
	var out []byte
	data := []byte("payload bytes")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out = msgp.AppendBytes(out[:0], data)
	}
	_ = out
}

