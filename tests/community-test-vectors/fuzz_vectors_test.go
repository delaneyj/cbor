package tests

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// FuzzCommunityVectors fuzzes around the community vectors by
// seeding the fuzzer with known-good CBOR payloads from appendix_a.json.
// The goal is to ensure that ValidateWellFormedBytes and DiagBytes do
// not panic on mutated inputs.
func FuzzCommunityVectors(f *testing.F) {
	root := "."
	appendix := filepath.Join(root, "appendix_a.json")
	b, err := os.ReadFile(appendix)
	if err == nil {
		var vects []struct {
			Hex        string `json:"hex"`
			Diagnostic string `json:"diagnostic"`
		}
		if err := json.Unmarshal(b, &vects); err == nil {
			for _, v := range vects {
				if v.Hex == "" {
					continue
				}
				msg, err := hex.DecodeString(v.Hex)
				if err == nil {
					f.Add(msg)
				}
			}
		}
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in community vectors fuzz: %v", r)
			}
		}()

		// ValidateWellFormedBytes should never panic.
		rest, err := cbor.ValidateWellFormedBytes(data)
		_ = rest
		if err != nil {
			return
		}
		// Only call DiagBytes on well-formed inputs.
		_, _, _ = cbor.DiagBytes(data)
	})
}

