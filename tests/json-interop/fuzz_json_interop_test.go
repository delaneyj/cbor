package tests

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

// FuzzJSONInterop fuzzes the JSON wrappers and general FromJSONBytes /
// ToJSONBytes round-trip to ensure they do not panic on arbitrary input.
func FuzzJSONInterop(f *testing.F) {
	seeds := []string{
		`{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"}`,
		`{"$base64":"QUJD"}`,
		`{"$base16":"000102"}`,
		`{"$base64url":"QUJD"}`,
		`{"$cbor":"gkFh"}`,
		`{"$uri":"https://example.com"}`,
		`{"$base64urlstr":"QUJD-_0"}`,
		`{"$base64str":"QUJD+w=="}`,
		`{"$regex":"^a+$"}`,
		`{"$mime":"Content-Type: text/plain\r\n\r\nhello"}`,
		`{"$selfdescribe":true}`,
		`{"$tag":99, "$":"text"}`,
		`{"$rfc3339":"2024-01-02T03:04:05Z"}`,
		`{"$epoch":1700000001.25}`,
		`[{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},{"$base64":"QUJD"}]`,
		`{"u":{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},"b":{"$base64":"QUJD"}}`,
		`{"t":{"$rfc3339":"2024-01-02T03:04:05Z"},"d":{"$decimal":[-2,"12345"]}}`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic in JSON interop fuzz: %v", r)
			}
		}()

		// FromJSONBytes should never panic. It may return an error for
		// invalid JSON or unsupported wrappers; those are ignored.
		b, err := cbor.FromJSONBytes(data)
		if err != nil {
			return
		}

		// The resulting bytes should be safe to validate as CBOR.
		if _, err := cbor.ValidateWellFormedBytes(b); err != nil {
			return
		}

		// ToJSONBytes should not panic for well-formed CBOR.
		_, _, _ = cbor.ToJSONBytes(b)
	})
}

