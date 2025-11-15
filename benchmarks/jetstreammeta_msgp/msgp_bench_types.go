package jetstreammeta_msgp

import (
	"fmt"

	js "github.com/delaneyj/cbor/tests/jetstreammeta"
)

// MsgpMetaSnapshot and its nested types mirror the structure of
// MetaSnapshot but are dedicated to tinylib/msgp code generation so
// that we can benchmark msgp's generated encoders without colliding
// with cborgen's helpers.

type MsgpMetaSnapshot struct {
	Streams []MsgpWriteableStreamAssignment `msg:"streams"`
}

type MsgpWriteableStreamAssignment struct {
	Client     *MsgpClientInfo                    `msg:"client,omitempty"`
	Created    int64                              `msg:"created"` // unix nanos
	ConfigJSON []byte                             `msg:"stream"`
	Group      *MsgpRaftGroup                     `msg:"group"`
	Sync       string                             `msg:"sync"`
	Consumers  []*MsgpWriteableConsumerAssignment `msg:"consumers,omitempty"`
}

type MsgpWriteableConsumerAssignment struct {
	Client     *MsgpClientInfo       `msg:"client,omitempty"`
	Created    int64                 `msg:"created"`
	Name       string                `msg:"name"`
	Stream     string                `msg:"stream"`
	ConfigJSON []byte                `msg:"consumer"`
	Group      *MsgpRaftGroup        `msg:"group"`
	State      *MsgpConsumerState    `msg:"state,omitempty"`
}

type MsgpClientInfo struct {
	Account string `msg:"acc,omitempty"`
	Service string `msg:"svc,omitempty"`
	Cluster string `msg:"cluster,omitempty"`
	RTTNano int64  `msg:"rtt,omitempty"`
}

type MsgpRaftGroup struct {
	Name      string   `msg:"name"`
	Peers     []string `msg:"peers"`
	Storage   int      `msg:"store"`
	Cluster   string   `msg:"cluster,omitempty"`
	Preferred string   `msg:"preferred,omitempty"`
	ScaleUp   bool     `msg:"scale_up,omitempty"`
}

type MsgpSequencePair struct {
	Consumer uint64 `msg:"consumer_seq"`
	Stream   uint64 `msg:"stream_seq"`
}

type MsgpPending struct {
	Sequence  uint64 `msg:"sequence"`
	Timestamp int64  `msg:"ts"`
}

type MsgpConsumerState struct {
	Delivered   MsgpSequencePair        `msg:"delivered"`
	AckFloor    MsgpSequencePair        `msg:"ack_floor"`
	Pending     map[string]*MsgpPending `msg:"pending,omitempty"`
	Redelivered map[string]uint64       `msg:"redelivered,omitempty"`
}

// ToMsgpMetaSnapshot converts the MetaSnapshot used for cborgen into
// the MsgpMetaSnapshot used for msgp benchmarks.
func ToMsgpMetaSnapshot(snap js.MetaSnapshot) MsgpMetaSnapshot {
	out := MsgpMetaSnapshot{Streams: make([]MsgpWriteableStreamAssignment, 0, len(snap.Streams))}
	for i := range snap.Streams {
		out.Streams = append(out.Streams, toMsgpStream(&snap.Streams[i]))
	}
	return out
}

func toMsgpStream(s *js.WriteableStreamAssignment) MsgpWriteableStreamAssignment {
	ms := MsgpWriteableStreamAssignment{
		Client:     toMsgpClient(s.Client),
		Created:    s.Created.UnixNano(),
		ConfigJSON: s.ConfigJSON,
		Group:      toMsgpGroup(s.Group),
		Sync:       s.Sync,
	}
	if len(s.Consumers) > 0 {
		ms.Consumers = make([]*MsgpWriteableConsumerAssignment, 0, len(s.Consumers))
		for _, ca := range s.Consumers {
			ms.Consumers = append(ms.Consumers, toMsgpConsumer(ca))
		}
	}
	return ms
}

func toMsgpClient(ci *js.ClientInfo) *MsgpClientInfo {
	if ci == nil {
		return nil
	}
	return &MsgpClientInfo{
		Account: ci.Account,
		Service: ci.Service,
		Cluster: ci.Cluster,
		RTTNano: int64(ci.RTT),
	}
}

func toMsgpGroup(rg *js.RaftGroup) *MsgpRaftGroup {
	if rg == nil {
		return nil
	}
	mg := &MsgpRaftGroup{
		Name:      rg.Name,
		Peers:     append([]string(nil), rg.Peers...),
		Storage:   int(rg.Storage),
		Cluster:   rg.Cluster,
		Preferred: rg.Preferred,
		ScaleUp:   rg.ScaleUp,
	}
	return mg
}

func toMsgpConsumer(ca *js.WriteableConsumerAssignment) *MsgpWriteableConsumerAssignment {
	if ca == nil {
		return nil
	}
	mc := &MsgpWriteableConsumerAssignment{
		Client:     toMsgpClient(ca.Client),
		Created:    ca.Created.UnixNano(),
		Name:       ca.Name,
		Stream:     ca.Stream,
		ConfigJSON: ca.ConfigJSON,
		Group:      toMsgpGroup(ca.Group),
	}
	if ca.State != nil {
		mc.State = toMsgpState(ca.State)
	}
	return mc
}

func toMsgpState(cs *js.ConsumerState) *MsgpConsumerState {
	ms := &MsgpConsumerState{
		Delivered: MsgpSequencePair{Consumer: cs.Delivered.Consumer, Stream: cs.Delivered.Stream},
		AckFloor:  MsgpSequencePair{Consumer: cs.AckFloor.Consumer, Stream: cs.AckFloor.Stream},
	}
	if len(cs.Pending) > 0 {
		ms.Pending = make(map[string]*MsgpPending, len(cs.Pending))
		for k, v := range cs.Pending {
			ms.Pending[fmt.Sprintf("%d", k)] = &MsgpPending{Sequence: v.Sequence, Timestamp: v.Timestamp}
		}
	}
	if len(cs.Redelivered) > 0 {
		ms.Redelivered = make(map[string]uint64, len(cs.Redelivered))
		for k, v := range cs.Redelivered {
			ms.Redelivered[fmt.Sprintf("%d", k)] = v
		}
	}
	return ms
}

