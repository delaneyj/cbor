package tests

import (
	"encoding/hex"
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

type rfcExample struct {
	name string
	diag string
	hex  string
}

var rfcExamples = []rfcExample{
	{
		name: "text-a",
		diag: "\"a\"",
		hex:  "6161",
	},
	{
		name: "zero",
		diag: "0",
		hex:  "00",
	},
	{
		name: "minus-one",
		diag: "-1",
		hex:  "20",
	},
	{
		name: "bytes-010203",
		diag: "h'010203'",
		hex:  "43010203",
	},
	{
		name: "array-1-2-3",
		diag: "[1, 2, 3]",
		hex:  "83010203",
	},
	{
		name: "map-a1-b2",
		diag: "{\"a\": 1, \"b\": 2}",
		hex:  "a2616101616202",
	},
	{
		name: "indef-array-1-2",
		diag: "[_ 1, 2]",
		hex:  "9f0102ff",
	},
	{
		name: "tag-epoch-datetime",
		diag: "1(1363896240)",
		hex:  "c11a514b67b0",
	},
}

func TestRFCExamplesDiagAndWellFormed(t *testing.T) {
	for _, ex := range rfcExamples {
		ex := ex
		t.Run(ex.name, func(t *testing.T) {
			msg, err := hex.DecodeString(ex.hex)
			if err != nil {
				t.Fatalf("bad hex %q: %v", ex.hex, err)
			}

			got, rest, err := cbor.DiagBytes(msg)
			if err != nil {
				t.Fatalf("DiagBytes error: %v", err)
			}
			if len(rest) != 0 {
				t.Fatalf("DiagBytes leftover: %d", len(rest))
			}
			if got != ex.diag {
				t.Fatalf("diag mismatch: got %q want %q (hex %s)", got, ex.diag, ex.hex)
			}

			rest2, err := cbor.ValidateWellFormedBytes(msg)
			if err != nil {
				t.Fatalf("ValidateWellFormedBytes error: %v", err)
			}
			if len(rest2) != 0 {
				t.Fatalf("ValidateWellFormedBytes leftover: %d", len(rest2))
			}
		})
	}
}

