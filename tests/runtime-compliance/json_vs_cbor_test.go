package tests

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
)

func TestIsLikelyJSON_Classification(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "json_object", data: []byte(`{"a":1}`), want: true},
		{name: "json_array", data: []byte(`[1,2,3]`), want: true},
		{name: "json_string", data: []byte(`"hi"`), want: true},
		{name: "json_number", data: []byte(`123`), want: true},
		{name: "json_bool", data: []byte(`true`), want: true},
		{name: "json_null", data: []byte(`null`), want: true},
		{name: "json_with_ws", data: []byte("  \n\t[1]"), want: true},
		{name: "invalid_utf8", data: []byte{0xff, 0xfe, 0xfd}, want: false},
		{name: "obvious_cbor_map", data: []byte{0xa1, 0x61, 0x61, 0x01}, want: false}, // {"a":1} in CBOR
		{name: "obvious_cbor_array", data: []byte{0x83, 0x01, 0x02, 0x03}, want: false}, // [1,2,3]
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := cbor.IsLikelyJSON(c.data)
			if got != c.want {
				t.Fatalf("IsLikelyJSON(%s) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

