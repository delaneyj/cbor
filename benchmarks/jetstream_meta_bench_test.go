package benchmarks

import (
	"testing"

	"github.com/delaneyj/cbor/tests/jetstreammeta"
)

// BenchmarkCBORRuntime_JetStreamMetaSnapshot_Encode exercises CBOR
// marshalling of a realistic JetStream meta snapshot workload. The
// fixture mirrors the shape and scale of the NATS
// BenchmarkJetStreamMetaSnapshot benchmark (200 streams, 500
// consumers each), but encodes the snapshot using this CBOR runtime
// instead of JSON+S2.
func BenchmarkCBORRuntime_JetStreamMetaSnapshot_Encode(b *testing.B) {
	snap := jetstreammeta.BuildMetaSnapshotFixture(
		jetstreammeta.DefaultNumStreams,
		jetstreammeta.DefaultNumConsumers,
	)

	// Sanity check that encoding succeeds once before benchmarking.
	if _, err := snap.MarshalCBOR(nil); err != nil {
		b.Fatalf("MarshalCBOR (warmup) failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var out []byte
	for i := 0; i < b.N; i++ {
		var err error
		out, err = snap.MarshalCBOR(out[:0])
		if err != nil {
			b.Fatalf("MarshalCBOR: %v", err)
		}
	}
	_ = out
}

