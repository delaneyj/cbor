package benchmarks

import (
	"testing"

	json "encoding/json"

	fxcbor "github.com/fxamacker/cbor/v2"
	msgp "github.com/tinylib/msgp/msgp"

	"github.com/delaneyj/cbor/tests/structs"
)

// benchPerson mirrors the fields of structs.Person but is defined
// locally so we can add tags for other libraries without impacting
// cborgen-driven code.
type benchPerson struct {
	Name string `json:"name" msg:"name"`
	Age  int    `json:"age" msg:"age"`
	Data []byte `json:"data" msg:"data"`
}

// newPerson constructs a sample structs.Person and its equivalent
// benchPerson value.
func newPerson() (structs.Person, benchPerson) {
	p := structs.Person{Name: "Alice", Age: 42, Data: []byte("hello world")}
	return p, benchPerson{Name: p.Name, Age: p.Age, Data: p.Data}
}

func BenchmarkCBORRuntime_Struct_Encode(b *testing.B) {
	p, _ := newPerson()
	b.ReportAllocs()
	b.ResetTimer()
	var out []byte
	for i := 0; i < b.N; i++ {
		out, _ = p.MarshalCBOR(out[:0])
	}
	_ = out
}

func BenchmarkCBORRuntime_Struct_DecodeSafe(b *testing.B) {
	p, _ := newPerson()
	enc, err := p.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Person
		if _, err := out.DecodeSafe(enc); err != nil {
			b.Fatalf("DecodeSafe: %v", err)
		}
	}
}

func BenchmarkCBORRuntime_Struct_DecodeTrusted(b *testing.B) {
	p, _ := newPerson()
	enc, err := p.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Person
		if _, err := out.DecodeTrusted(enc); err != nil {
			b.Fatalf("DecodeTrusted: %v", err)
		}
	}
}

func BenchmarkCBORRuntime_Scalars_Encode(b *testing.B) {
	// Use the Scalars fixture from tests/structs to exercise a broad
	// set of field types in a single struct.
	s := structs.Scalars{
		S:     "s",
		B:     true,
		I:     1,
		I8:    2,
		I16:   3,
		I32:   4,
		I64:   5,
		U:     6,
		U8:    7,
		U16:   8,
		U32:   9,
		U64:   10,
		F32:   11.5,
		F64:   12.25,
		Data:  []byte("payload"),
		Ints:  []int{1, 2, 3, 4},
		Names: []string{"a", "b", "c"},
		Scores: map[string]int{"x": 1, "y": 2},
	}
	b.ReportAllocs()
	b.ResetTimer()
	var out []byte
	for i := 0; i < b.N; i++ {
		var err error
		out, err = s.MarshalCBOR(out[:0])
		if err != nil {
			b.Fatalf("MarshalCBOR: %v", err)
		}
	}
	_ = out
}

func BenchmarkCBORRuntime_Scalars_DecodeSafe(b *testing.B) {
	s := structs.Scalars{S: "s", B: true, I: 1}
	enc, err := s.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Scalars
		if _, err := out.DecodeSafe(enc); err != nil {
			b.Fatalf("DecodeSafe: %v", err)
		}
	}
}

func BenchmarkCBORRuntime_Scalars_DecodeTrusted(b *testing.B) {
	s := structs.Scalars{S: "s", B: true, I: 1}
	enc, err := s.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Scalars
		if _, err := out.DecodeTrusted(enc); err != nil {
			b.Fatalf("DecodeTrusted: %v", err)
		}
	}
}

func BenchmarkCBORRuntime_Containers_Encode(b *testing.B) {
	c := structs.Containers{
		Items: []structs.Scalars{{S: "a", I: 1}, {S: "b", I: 2}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	var out []byte
	for i := 0; i < b.N; i++ {
		var err error
		out, err = c.MarshalCBOR(out[:0])
		if err != nil {
			b.Fatalf("MarshalCBOR: %v", err)
		}
	}
	_ = out
}

func BenchmarkCBORRuntime_Containers_DecodeSafe(b *testing.B) {
	c := structs.Containers{Items: []structs.Scalars{{S: "a", I: 1}}}
	enc, err := c.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Containers
		if _, err := out.DecodeSafe(enc); err != nil {
			b.Fatalf("DecodeSafe: %v", err)
		}
	}
}

func BenchmarkCBORRuntime_Containers_DecodeTrusted(b *testing.B) {
	c := structs.Containers{Items: []structs.Scalars{{S: "a", I: 1}}}
	enc, err := c.MarshalCBOR(nil)
	if err != nil {
		b.Fatalf("MarshalCBOR: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out structs.Containers
		if _, err := out.DecodeTrusted(enc); err != nil {
			b.Fatalf("DecodeTrusted: %v", err)
		}
	}
}

func BenchmarkFXCBOR_Struct_Encode(b *testing.B) {
	_, bp := newPerson()
	encMode, err := fxcbor.CanonicalEncOptions().EncMode()
	if err != nil {
		b.Fatalf("fxcbor EncMode: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var out []byte
	for i := 0; i < b.N; i++ {
		out, err = encMode.Marshal(bp)
		if err != nil {
			b.Fatalf("fxcbor Marshal: %v", err)
		}
	}
	_ = out
}

func BenchmarkFXCBOR_Struct_Decode(b *testing.B) {
	_, bp := newPerson()
	encMode, err := fxcbor.CanonicalEncOptions().EncMode()
	if err != nil {
		b.Fatalf("fxcbor EncMode: %v", err)
	}
	decMode, err := fxcbor.DecOptions{}.DecMode()
	if err != nil {
		b.Fatalf("fxcbor DecMode: %v", err)
	}
	enc, err := encMode.Marshal(bp)
	if err != nil {
		b.Fatalf("fxcbor Marshal: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchPerson
		if err := decMode.Unmarshal(enc, &out); err != nil {
			b.Fatalf("fxcbor Unmarshal: %v", err)
		}
	}
}

func BenchmarkJSONv1_Struct_Encode(b *testing.B) {
	_, bp := newPerson()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(bp); err != nil {
			b.Fatalf("json.Marshal: %v", err)
		}
	}
}

func BenchmarkJSONv1_Struct_Decode(b *testing.B) {
	_, bp := newPerson()
	enc, err := json.Marshal(bp)
	if err != nil {
		b.Fatalf("json.Marshal: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchPerson
		if err := json.Unmarshal(enc, &out); err != nil {
			b.Fatalf("json.Unmarshal: %v", err)
		}
	}
}

func BenchmarkMsgp_Struct_Encode(b *testing.B) {
	_, bp := newPerson()
	m := map[string]any{"name": bp.Name, "age": bp.Age, "data": bp.Data}
	b.ReportAllocs()
	b.ResetTimer()
	var out []byte
	for i := 0; i < b.N; i++ {
		var err error
		out, err = msgp.AppendIntf(out[:0], m)
		if err != nil {
			b.Fatalf("msgp MarshalMsg: %v", err)
		}
	}
	_ = out
}

// For msgp decode we currently focus on encode performance; decode
// requires either generated methods or additional reflection helpers.
// Benchmarks here are intended primarily to compare encode-side costs
// for a representative struct.
