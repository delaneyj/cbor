package tests

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	cbor "github.com/delaneyj/cbor/runtime"
)

func TestTag1_Time_IntAndFloat(t *testing.T) {
	// Integer seconds
	ti := time.Unix(1700000000, 0).UTC()
	b := cbor.AppendTime(nil, ti)
	got, rest, err := cbor.ReadTimeBytes(b)
	if err != nil {
		t.Fatalf("int time read err: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("int time rest: %d", len(rest))
	}
	if !got.Equal(ti) {
		t.Fatalf("int time mismatch: got %v want %v", got, ti)
	}

	// Fractional seconds
	tf := time.Unix(1700000001, 123_456_789).UTC()
	b = cbor.AppendTime(nil, tf)
	got, rest, err = cbor.ReadTimeBytes(b)
	if err != nil {
		t.Fatalf("float time read err: %v", err)
	}
	if len(rest) != 0 {
		t.Fatalf("float time rest: %d", len(rest))
	}
	// Allow small rounding error due to float seconds representation
	dt := got.Sub(tf)
	if dt < 0 {
		dt = -dt
	}
	if dt > time.Microsecond {
		t.Fatalf("float time mismatch: got %v want %v delta=%v", got, tf, dt)
	}
}

func TestBase64TextTags(t *testing.T) {
	sURL := "QUJD-_0"
	sStd := "QUJD+w=="
	b := cbor.AppendBase64URLString(nil, sURL)
	gotS, rest, err := cbor.ReadBase64URLStringBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("base64url text err: %v rest:%d", err, len(rest))
	}
	if gotS != sURL {
		t.Fatalf("base64url text mismatch: %q != %q", gotS, sURL)
	}

	b = cbor.AppendBase64String(nil, sStd)
	gotS, rest, err = cbor.ReadBase64StringBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("base64 text err: %v rest:%d", err, len(rest))
	}
	if gotS != sStd {
		t.Fatalf("base64 text mismatch: %q != %q", gotS, sStd)
	}
}

func TestURIRegexMIME(t *testing.T) {
	uri := "https://example.com/a?b=c"
	b := cbor.AppendURI(nil, uri)
	u, rest, err := cbor.ReadURIStringBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("uri err: %v rest:%d", err, len(rest))
	}
	if u != uri {
		t.Fatalf("uri mismatch: %q != %q", u, uri)
	}

	pat := "^a+b?$"
	b = cbor.AppendRegexpString(nil, pat)
	ps, rest, err := cbor.ReadRegexpStringBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("regex str err: %v rest:%d", err, len(rest))
	}
	if ps != pat {
		t.Fatalf("regex str mismatch: %q != %q", ps, pat)
	}

	r := regexp.MustCompile(pat)
	b = cbor.AppendRegexp(nil, r)
	rr, rest, err := cbor.ReadRegexpBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("regex err: %v rest:%d", err, len(rest))
	}
	if !rr.MatchString("aa") || rr.MatchString("ac") {
		t.Fatalf("regex behavior unexpected")
	}

	mime := "Content-Type: text/plain\r\n\r\nhello"
	b = cbor.AppendMIMEString(nil, mime)
	ms, rest, err := cbor.ReadMIMEStringBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("mime err: %v rest:%d", err, len(rest))
	}
	if ms != mime {
		t.Fatalf("mime mismatch: %q != %q", ms, mime)
	}
}

func TestUUIDEmbeddedSelfDescribe(t *testing.T) {
	var id [16]byte
	for i := 0; i < 16; i++ {
		id[i] = byte(i)
	}
	b := cbor.AppendUUID(nil, id)
	got, rest, err := cbor.ReadUUIDBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("uuid err: %v rest:%d", err, len(rest))
	}
	if got != id {
		t.Fatalf("uuid mismatch")
	}

	// Embedded CBOR
	payload := cbor.AppendInt64(nil, 42)
	b = cbor.AppendEmbeddedCBOR(nil, payload)
	pb, rest, err := cbor.ReadEmbeddedCBORBytes(b)
	if err != nil || len(rest) != 0 {
		t.Fatalf("embedded err: %v rest:%d", err, len(rest))
	}
	if !bytes.Equal(pb, payload) {
		t.Fatalf("embedded payload mismatch")
	}

	// Self-describe
	b = cbor.AppendSelfDescribeCBOR(nil)
	b = cbor.AppendMapHeader(b, 1)
	b = cbor.AppendString(b, "a")
	b = cbor.AppendInt64(b, 1)
	r, found, err := cbor.StripSelfDescribeCBOR(b)
	if err != nil || !found {
		t.Fatalf("self-describe strip err:%v found:%v", err, found)
	}
	sz, r, err := cbor.ReadMapHeaderBytes(r)
	if err != nil || sz != 1 {
		t.Fatalf("map after self-describe err:%v sz:%d", err, sz)
	}
}

func TestTagNegativeCases(t *testing.T) {
	// Wrong tag for UUID should fail.
	// Encode tag(32) with a 16-byte payload and attempt to read as UUID.
	var id [16]byte
	for i := 0; i < 16; i++ {
		id[i] = byte(i)
	}
	b := cbor.AppendTag(nil, 32)
	b = cbor.AppendBytes(b, id[:])
	_, _, err := cbor.ReadUUIDBytes(b)
	if err == nil {
		t.Fatalf("expected error for wrong UUID tag, got nil")
	}

	// UUID with wrong payload length should fail.
	short := cbor.AppendUUID(nil, id)
	// Truncate one byte from the payload.
	short = short[:len(short)-1]
	_, _, err = cbor.ReadUUIDBytes(short)
	if err == nil {
		t.Fatalf("expected error for short UUID payload, got nil")
	}

	// Decimal fraction with wrong array length should fail with error.
	// Build tag(4) with array of length 1.
	decBad := cbor.AppendTag(nil, 4)
	decBad = cbor.AppendArrayHeader(decBad, 1)
	decBad = cbor.AppendInt64(decBad, 0)
	if _, _, _, err := cbor.ReadDecimalFractionBytes(decBad); err == nil {
		t.Fatalf("expected error for malformed decimal fraction, got nil")
	}

	// Bigfloat with non-integer mantissa should fail.
	bigBad := cbor.AppendTag(nil, 5)
	bigBad = cbor.AppendArrayHeader(bigBad, 2)
	bigBad = cbor.AppendInt64(bigBad, 0)
	// Second element: text instead of integer/bignum.
	bigBad = cbor.AppendString(bigBad, "not-an-int")
	if _, _, _, err := cbor.ReadBigfloatBytes(bigBad); err == nil {
		t.Fatalf("expected error for malformed bigfloat, got nil")
	}
}
