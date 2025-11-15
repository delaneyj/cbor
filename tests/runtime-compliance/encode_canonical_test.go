package tests

import (
	"encoding/hex"
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

func TestCanonicalIntEncoding(t *testing.T) {
	cases := []struct {
		name   string
		build  func() []byte
		wantHex string
	}{
		{
			name: "int_0",
			build: func() []byte { return cbor.AppendInt64(nil, 0) },
			wantHex: "00",
		},
		{
			name: "int_1",
			build: func() []byte { return cbor.AppendInt64(nil, 1) },
			wantHex: "01",
		},
		{
			name: "int_10",
			build: func() []byte { return cbor.AppendInt64(nil, 10) },
			wantHex: "0a",
		},
		{
			name: "int_23",
			build: func() []byte { return cbor.AppendInt64(nil, 23) },
			wantHex: "17",
		},
		{
			name: "int_24",
			build: func() []byte { return cbor.AppendInt64(nil, 24) },
			wantHex: "1818",
		},
		{
			name: "int_255",
			build: func() []byte { return cbor.AppendInt64(nil, 255) },
			wantHex: "18ff",
		},
		{
			name: "int_256",
			build: func() []byte { return cbor.AppendInt64(nil, 256) },
			wantHex: "190100",
		},
		{
			name: "neg_1",
			build: func() []byte { return cbor.AppendInt64(nil, -1) },
			wantHex: "20",
		},
		{
			name: "neg_10",
			build: func() []byte { return cbor.AppendInt64(nil, -10) },
			wantHex: "29",
		},
		{
			name: "neg_24",
			build: func() []byte { return cbor.AppendInt64(nil, -24) },
			wantHex: "37",
		},
		{
			name: "neg_25",
			build: func() []byte { return cbor.AppendInt64(nil, -25) },
			wantHex: "3818",
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			b := c.build()
			got := hex.EncodeToString(b)
			if got != c.wantHex {
				t.Fatalf("canonical int encoding mismatch: got %s want %s", got, c.wantHex)
			}
		})
	}
}

func TestCanonicalFloatEncoding(t *testing.T) {
	// 1.0 should be encoded as float16 (0xf9 ...)
	b := cbor.AppendFloatCanonical(nil, 1.0)
	if len(b) != 3 || b[0] != 0xf9 {
		t.Fatalf("1.0 not encoded as float16, got %x", b)
	}

	// A value not exactly representable in float32 (e.g. 1/3) should use float64 (0xfb ...).
	val := 1.0 / 3.0
	b = cbor.AppendFloatCanonical(nil, val)
	if len(b) != 9 || b[0] != 0xfb {
		t.Fatalf("1/3 not encoded as float64, got %x", b)
	}
}

