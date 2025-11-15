package benchmarks

import (
	"testing"

	cbor "github.com/delaneyj/cbor/runtime"
	msgp "github.com/tinylib/msgp/msgp"
)

// TestData mirrors the prototype's benchmark_comparison.go payload so we
// exercise the exact same shapes and primitive paths for both the CBOR
// runtime and tinylib/msgp in a table-driven fashion.
type TestData struct {
	Name    string
	Age     int64
	Email   string
	Active  bool
	Balance float64
	Tags    []string
	Scores  map[string]int64
}

func encodeMsgpTestData(data TestData) []byte {
	var buf []byte
	buf = msgp.AppendString(buf, data.Name)
	buf = msgp.AppendInt64(buf, data.Age)
	buf = msgp.AppendString(buf, data.Email)
	buf = msgp.AppendBool(buf, data.Active)
	buf = msgp.AppendFloat64(buf, data.Balance)

	buf = msgp.AppendArrayHeader(buf, uint32(len(data.Tags)))
	for _, tag := range data.Tags {
		buf = msgp.AppendString(buf, tag)
	}

	buf = msgp.AppendMapHeader(buf, uint32(len(data.Scores)))
	for k, v := range data.Scores {
		buf = msgp.AppendString(buf, k)
		buf = msgp.AppendInt64(buf, v)
	}

	return buf
}

func encodeCBORTestData(data TestData) []byte {
	var buf []byte
	buf = cbor.AppendString(buf, data.Name)
	buf = cbor.AppendInt64(buf, data.Age)
	buf = cbor.AppendString(buf, data.Email)
	buf = cbor.AppendBool(buf, data.Active)
	buf = cbor.AppendFloat64(buf, data.Balance)

	buf = cbor.AppendArrayHeader(buf, uint32(len(data.Tags)))
	for _, tag := range data.Tags {
		buf = cbor.AppendString(buf, tag)
	}

	buf = cbor.AppendMapHeader(buf, uint32(len(data.Scores)))
	for k, v := range data.Scores {
		buf = cbor.AppendString(buf, k)
		buf = cbor.AppendInt64(buf, v)
	}

	return buf
}

func decodeMsgpTestData(b []byte) error {
	buf := b
	var err error

	// Scalars
	_, buf, err = msgp.ReadStringBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = msgp.ReadInt64Bytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = msgp.ReadStringBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = msgp.ReadBoolBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = msgp.ReadFloat64Bytes(buf)
	if err != nil {
		return err
	}

	// Tags array
	var arrSize uint32
	arrSize, buf, err = msgp.ReadArrayHeaderBytes(buf)
	if err != nil {
		return err
	}
	for j := uint32(0); j < arrSize; j++ {
		_, buf, err = msgp.ReadStringBytes(buf)
		if err != nil {
			return err
		}
	}

	// Scores map
	var mapSize uint32
	mapSize, buf, err = msgp.ReadMapHeaderBytes(buf)
	if err != nil {
		return err
	}
	for j := uint32(0); j < mapSize; j++ {
		_, buf, err = msgp.ReadStringBytes(buf)
		if err != nil {
			return err
		}
		_, buf, err = msgp.ReadInt64Bytes(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func decodeCBORTestData(b []byte) error {
	buf := b
	var err error

	// Scalars
	_, buf, err = cbor.ReadStringBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = cbor.ReadInt64Bytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = cbor.ReadStringBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = cbor.ReadBoolBytes(buf)
	if err != nil {
		return err
	}
	_, buf, err = cbor.ReadFloat64Bytes(buf)
	if err != nil {
		return err
	}

	// Tags array
	var arrSize uint32
	arrSize, buf, err = cbor.ReadArrayHeaderBytes(buf)
	if err != nil {
		return err
	}
	for j := uint32(0); j < arrSize; j++ {
		_, buf, err = cbor.ReadStringBytes(buf)
		if err != nil {
			return err
		}
	}

	// Scores map
	var mapSize uint32
	mapSize, buf, err = cbor.ReadMapHeaderBytes(buf)
	if err != nil {
		return err
	}
	for j := uint32(0); j < mapSize; j++ {
		_, buf, err = cbor.ReadStringBytes(buf)
		if err != nil {
			return err
		}
		_, buf, err = cbor.ReadInt64Bytes(buf)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestTestDataPrimitivePathsParity(t *testing.T) {
	data := TestData{
		Name:    "Alice Johnson",
		Age:     30,
		Email:   "alice@example.com",
		Active:  true,
		Balance: 12345.67,
		Tags:    []string{"premium", "verified", "active"},
		Scores:  map[string]int64{"math": 95, "science": 88, "history": 92},
	}

	cases := []struct {
		name string
		enc  func(TestData) []byte
		dec  func([]byte) error
	}{
		{"msgp", encodeMsgpTestData, decodeMsgpTestData},
		{"cbor", encodeCBORTestData, decodeCBORTestData},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.enc(data)
			if len(b) == 0 {
				t.Fatalf("%s: empty encoding", tc.name)
			}
			if err := tc.dec(b); err != nil {
				t.Fatalf("%s: decode err: %v", tc.name, err)
			}
		})
	}
}
