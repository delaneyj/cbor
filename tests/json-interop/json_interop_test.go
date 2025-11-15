package tests

import (
	"encoding/json"
	"testing"
	"time"

	cbor "github.com/delaneyj/cbor/runtime"
)

func TestJSONInterop_Wrappers(t *testing.T) {
	cases := []struct {
		name string
		js   string
		want string
	}{
		{
			name: "uuid",
			js:   `{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"}`,
			want: `{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"}`,
		},
		{
			name: "base64-bytes",
			js:   `{"$base64":"QUJD"}`,
			want: `{"$base64":"QUJD"}`,
		},
		{
			name: "base16-bytes",
			js:   `{"$base16":"000102"}`,
			want: `{"$base16":"000102"}`,
		},
		{
			name: "base64url-bytes",
			js:   `{"$base64url":"QUJD"}`,
			want: `{"$base64url":"QUJD"}`,
		},
		{
			name: "embedded-cbor",
			js:   `{"$cbor":"gkFh"}`,
			want: `{"$cbor":"gkFh"}`,
		},
		{
			name: "uri",
			js:   `{"$uri":"https://example.com"}`,
			// ToJSONBytes unwraps URI tags into plain JSON strings.
			want: `"https://example.com"`,
		},
		{
			name: "base64urlstr",
			js:   `{"$base64urlstr":"QUJD-_0"}`,
			want: `{"$base64urlstr":"QUJD-_0"}`,
		},
		{
			name: "base64str",
			js:   `{"$base64str":"QUJD+w=="}`,
			want: `{"$base64str":"QUJD+w=="}`,
		},
		{
			name: "regex",
			js:   `{"$regex":"^a+$"}`,
			want: `{"$regex":"^a+$"}`,
		},
		{
			name: "mime",
			js:   `{"$mime":"Content-Type: text/plain\r\n\r\nhello"}`,
			want: `{"$mime":"Content-Type: text/plain\r\n\r\nhello"}`,
		},
		{
			name: "selfdescribe",
			js:   `{"$selfdescribe":true}`,
			want: `{"$selfdescribe":true}`,
		},
		{
			name: "generic-tag",
			js:   `{"$tag":99, "$":"text"}`,
			want: `{"$tag":99,"$":"text"}`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			b, err := cbor.FromJSONBytes([]byte(c.js))
			if err != nil {
				t.Fatalf("FromJSONBytes(%s) err: %v", c.js, err)
			}
			out, _, err := cbor.ToJSONBytes(b)
			if err != nil {
				t.Fatalf("ToJSONBytes for %s err: %v", c.js, err)
			}
			gotNorm := normalizeJSON(out)
			wantNorm := normalizeJSON([]byte(c.want))
			if gotNorm != wantNorm {
				t.Fatalf("json round-trip mismatch:\n got: %s\nwant: %s", gotNorm, wantNorm)
			}
		})
	}
}

func TestJSONInterop_Times(t *testing.T) {
	t0 := time.Date(2024, 1, 2, 3, 4, 5, 6, time.UTC).Format(time.RFC3339Nano)
	js0 := `{"$rfc3339":` + string(mustJSON(t0)) + `}`
	b0, err := cbor.FromJSONBytes([]byte(js0))
	if err != nil {
		t.Fatalf("$rfc3339 FromJSONBytes err: %v", err)
	}
	out0, _, err := cbor.ToJSONBytes(b0)
	if err != nil {
		t.Fatalf("$rfc3339 ToJSONBytes err: %v", err)
	}
	got0 := normalizeJSON(out0)
	want0 := normalizeJSON(mustJSON(t0))
	if got0 != want0 {
		t.Fatalf("$rfc3339 canonical json mismatch:\n got: %s\nwant: %s", got0, want0)
	}

	js1 := `{"$epoch":1700000001.25}`
	b1, err := cbor.FromJSONBytes([]byte(js1))
	if err != nil {
		t.Fatalf("$epoch FromJSONBytes err: %v", err)
	}
	out1, _, err := cbor.ToJSONBytes(b1)
	if err != nil {
		t.Fatalf("$epoch ToJSONBytes err: %v", err)
	}
	// $epoch is rendered as an RFC3339 string in the local timezone.
	// Decode the JSON string and ensure it represents the expected instant.
	var s string
	if err := json.Unmarshal(out1, &s); err != nil {
		t.Fatalf("$epoch ToJSONBytes produced non-string JSON: %v", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("$epoch ToJSONBytes produced invalid RFC3339: %v", err)
	}
	expected := time.Unix(1700000001, int64(0.25*1e9)).UTC()
	if !parsed.Equal(expected) {
		t.Fatalf("$epoch instant mismatch: got %v want %v", parsed, expected)
	}
}

func TestJSONInterop_DecimalAndBigfloat(t *testing.T) {
	cases := []struct {
		name string
		js   string
		want string
	}{
		{
			name: "decimal",
			js:   `{"$decimal":[-2,"12345"]}`,
			want: `{"$decimal":[-2,"12345"]}`,
		},
		{
			name: "bigfloat",
			js:   `{"$bigfloat":[10,"12345"]}`,
			want: `{"$bigfloat":[10,"12345"]}`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			b, err := cbor.FromJSONBytes([]byte(c.js))
			if err != nil {
				t.Fatalf("FromJSONBytes(%s) err: %v", c.js, err)
			}
			out, _, err := cbor.ToJSONBytes(b)
			if err != nil {
				t.Fatalf("ToJSONBytes for %s err: %v", c.js, err)
			}
			gotNorm := normalizeJSON(out)
			wantNorm := normalizeJSON([]byte(c.want))
			if gotNorm != wantNorm {
				t.Fatalf("json round-trip mismatch:\n got: %s\nwant: %s", gotNorm, wantNorm)
			}
		})
	}
}

func TestJSONInterop_NestedWrappers(t *testing.T) {
	cases := []struct {
		name string
		js   string
		want string
	}{
		{
			name: "array_of_wrappers",
			js: `[{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},{"$base64":"QUJD"}]`,
			want: `[{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},{"$base64":"QUJD"}]`,
		},
		{
			name: "map_of_wrappers",
			js: `{"u":{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},"b":{"$base64":"QUJD"}}`,
			want: `{"u":{"$uuid":"00112233-4455-6677-8899-aabbccddeeff"},"b":{"$base64":"QUJD"}}`,
		},
		{
			name: "mixed_time_and_bytes",
			js: `{"t":{"$rfc3339":"2024-01-02T03:04:05Z"},"d":{"$decimal":[-2,"12345"]}}`,
			want: `{"t":"2024-01-02T03:04:05Z","d":{"$decimal":[-2,"12345"]}}`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			b, err := cbor.FromJSONBytes([]byte(c.js))
			if err != nil {
				t.Fatalf("FromJSONBytes(%s) err: %v", c.js, err)
			}
			out, _, err := cbor.ToJSONBytes(b)
			if err != nil {
				t.Fatalf("ToJSONBytes for %s err: %v", c.js, err)
			}
			gotNorm := normalizeJSON(out)
			wantNorm := normalizeJSON([]byte(c.want))
			if gotNorm != wantNorm {
				t.Fatalf("nested json round-trip mismatch:\n got: %s\nwant: %s", gotNorm, wantNorm)
			}
		})
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func normalizeJSON(b []byte) string {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return string(b)
	}
	return string(out)
}
