package jetstreammeta

import (
	"encoding/json"
	"testing"
	"time"
)

func TestClientInfo_Encode(t *testing.T) {
	ci := &ClientInfo{Account: "G", Service: "JS", Cluster: "R3S"}
	if _, err := ci.MarshalCBOR(nil); err != nil {
		t.Fatalf("ClientInfo.MarshalCBOR failed: %v", err)
	}
}

func TestRaftGroup_Encode(t *testing.T) {
	rg := &RaftGroup{Name: "rg", Peers: []string{"n1", "n2"}, Storage: MemoryStorage}
	if _, err := rg.MarshalCBOR(nil); err != nil {
		t.Fatalf("raftGroup.MarshalCBOR failed: %v", err)
	}
}

func TestWriteableConsumerAssignment_Encode(t *testing.T) {
	cfgJSON, _ := json.Marshal(ConsumerConfigSnapshot{Durable: "C", MemoryStorage: true})
	ca := &WriteableConsumerAssignment{
		Created:    testTime(),
		Name:       "C",
		Stream:     "S",
		ConfigJSON: json.RawMessage(cfgJSON),
	}
	if _, err := ca.MarshalCBOR(nil); err != nil {
		t.Fatalf("WriteableConsumerAssignment.MarshalCBOR failed: %v", err)
	}
}

func TestWriteableStreamAssignment_Encode(t *testing.T) {
	ci := &ClientInfo{Account: "G", Service: "JS", Cluster: "R3S"}
	rg := &RaftGroup{Name: "rg", Peers: []string{"n1", "n2"}, Storage: MemoryStorage}
	cfgJSON, _ := json.Marshal(StreamConfigSnapshot{Name: "S", Subjects: []string{"SUB"}, Storage: MemoryStorage})
	wa := &WriteableStreamAssignment{
		Client:     ci,
		Created:    testTime(),
		ConfigJSON: json.RawMessage(cfgJSON),
		Group:      rg,
		Sync:       "_INBOX.sync",
	}
	if _, err := wa.MarshalCBOR(nil); err != nil {
		t.Fatalf("WriteableStreamAssignment.MarshalCBOR failed: %v", err)
	}
}

func TestMetaSnapshot_Encode_DoesNotPanic(t *testing.T) {
	ci := &ClientInfo{Account: "G", Service: "JS", Cluster: "R3S"}
	rg := &RaftGroup{Name: "rg", Peers: []string{"n1", "n2"}, Storage: MemoryStorage}
	cfgJSON, _ := json.Marshal(StreamConfigSnapshot{Name: "S", Subjects: []string{"SUB"}, Storage: MemoryStorage})
	ccfgJSON, _ := json.Marshal(ConsumerConfigSnapshot{Durable: "C", MemoryStorage: true})
	ca := &WriteableConsumerAssignment{
		Client:     ci,
		Created:    testTime(),
		Name:       "C",
		Stream:     "S",
			ConfigJSON: json.RawMessage(ccfgJSON),
		Group:      rg,
			State: &ConsumerState{
				Delivered: SequencePair{Consumer: 1, Stream: 1},
				AckFloor:  SequencePair{Consumer: 0, Stream: 0},
				Pending: map[uint64]*Pending{
					1: {Sequence: 1, Timestamp: testTime().UnixNano()},
				},
				Redelivered: map[uint64]uint64{1: 2},
			},
	}
	ws := WriteableStreamAssignment{
		Client:     ci,
		Created:    testTime(),
		ConfigJSON: json.RawMessage(cfgJSON),
		Group:      rg,
		Sync:       "_INBOX.sync",
		Consumers:  []*WriteableConsumerAssignment{ca},
	}
	snap := MetaSnapshot{Streams: []WriteableStreamAssignment{ws}}
	if _, err := snap.MarshalCBOR(nil); err != nil {
		t.Fatalf("MetaSnapshot.MarshalCBOR failed: %v", err)
	}
}

func TestBuildMetaSnapshotFixture_Encode(t *testing.T) {
	snap := BuildMetaSnapshotFixture(2, 2)
	if _, err := snap.MarshalCBOR(nil); err != nil {
		t.Fatalf("BuildMetaSnapshotFixture MarshalCBOR failed: %v", err)
	}
}

func TestBuildMetaSnapshotFixture_TrustedDecode(t *testing.T) {
	orig := BuildMetaSnapshotFixture(2, 2)
	b, err := orig.MarshalCBOR(nil)
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	var out MetaSnapshot
	if _, err := out.DecodeTrusted(b); err != nil {
		t.Fatalf("DecodeTrusted: %v", err)
	}
}

func testTime() time.Time { return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) }
