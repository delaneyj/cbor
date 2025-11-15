package jetstreammeta

import (
	"os"
	"strings"
	"testing"
)

// TestGeneratedMarshalCBOR_NoAppendInterface verifies that the hot
// JetStream snapshot structs use only inline encode logic (map/array
// headers, primitive appends, and MarshalCBOR calls) and do not fall
// back to the dynamic AppendInterface helper in their MarshalCBOR
// implementations. This mirrors the msgp-style encode structure.
func TestGeneratedMarshalCBOR_NoAppendInterface(t *testing.T) {
	src, err := os.ReadFile("types_cbor.go")
	if err != nil {
		t.Fatalf("read types_cbor.go: %v", err)
	}

	file := string(src)

	checkNoAppendInterface := func(typeName string) {
		sig := "func (x *" + typeName + ") MarshalCBOR"
		start := strings.Index(file, sig)
		if start == -1 {
			t.Fatalf("did not find MarshalCBOR for %s", typeName)
		}
		// Heuristic: scan until the closing brace of this function, which
		// is followed either by a blank line or the next comment.
		end := strings.Index(file[start:], "\n}\n")
		if end == -1 {
			t.Fatalf("could not find end of MarshalCBOR for %s", typeName)
		}
		body := file[start : start+end]
		if strings.Contains(body, "AppendInterface(") {
			t.Fatalf("MarshalCBOR for %s uses AppendInterface; expected only inline encode helpers", typeName)
		}
	}

	checkNoAppendInterface("MetaSnapshot")
	checkNoAppendInterface("WriteableStreamAssignment")
	checkNoAppendInterface("WriteableConsumerAssignment")
	checkNoAppendInterface("ConsumerState")
}

