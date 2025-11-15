package jetstreammeta

// This package defines a reduced, self-contained model of the
// JetStream meta snapshot structures used by
// github.com/nats-io/nats-server/v2 in jetstream_cluster.go.
//
// It is intentionally trimmed down to only the fields that
// participate in snapshot marshalling so that we can benchmark
// CBOR encoding/decoding of a realistic, highly nested object
// graph without depending on the NATS server codebase itself.

import (
	"encoding/json"
	"time"

	cbor "github.com/delaneyj/cbor/runtime"
)

// StorageType determines how messages are stored for retention.
// These values mirror the identifiers used by NATS.
type StorageType int

const (
	// FileStorage stores data on disk.
	FileStorage = StorageType(22)
	// MemoryStorage stores data in memory only.
	MemoryStorage = StorageType(33)
)

// MarshalCBOR encodes the storage type as a small integer code so that
// generated code can delegate to this helper instead of inlining the
// encoding logic in multiple places.
func (st StorageType) MarshalCBOR(b []byte) ([]byte, error) {
	return cbor.AppendInt(b, int(st)), nil
}

// UnmarshalCBOR decodes a storage type integer code.
func (st *StorageType) UnmarshalCBOR(b []byte) ([]byte, error) {
	val, rest, err := cbor.ReadIntBytes(b)
	if err != nil {
		return b, err
	}
	*st = StorageType(val)
	return rest, nil
}

// ClientInfo is a reduced copy of the NATS ClientInfo struct with
// only JSON/Cbor-visible fields retained. The Tags field is
// simplified to []string to avoid pulling in external dependencies.
type ClientInfo struct {
	Start      *time.Time    `json:"start,omitempty" msg:"start,omitempty"`
	Host       string        `json:"host,omitempty" msg:"host,omitempty"`
	ID         uint64        `json:"id,omitempty" msg:"id,omitempty"`
	Account    string        `json:"acc,omitempty" msg:"acc,omitempty"`
	Service    string        `json:"svc,omitempty" msg:"svc,omitempty"`
	User       string        `json:"user,omitempty" msg:"user,omitempty"`
	Name       string        `json:"name,omitempty" msg:"name,omitempty"`
	Lang       string        `json:"lang,omitempty" msg:"lang,omitempty"`
	Version    string        `json:"ver,omitempty" msg:"ver,omitempty"`
	RTT        time.Duration `json:"rtt,omitempty" msg:"rtt,omitempty"`
	Server     string        `json:"server,omitempty" msg:"server,omitempty"`
	Cluster    string        `json:"cluster,omitempty" msg:"cluster,omitempty"`
	Alternates []string      `json:"alts,omitempty" msg:"alts,omitempty"`
	Stop       *time.Time    `json:"stop,omitempty" msg:"stop,omitempty"`
	Jwt        string        `json:"jwt,omitempty" msg:"jwt,omitempty"`
	IssuerKey  string        `json:"issuer_key,omitempty" msg:"issuer_key,omitempty"`
	NameTag    string        `json:"name_tag,omitempty" msg:"name_tag,omitempty"`
	Tags       []string      `json:"tags,omitempty" msg:"tags,omitempty"`
	Kind       string        `json:"kind,omitempty" msg:"kind,omitempty"`
	ClientType string        `json:"client_type,omitempty" msg:"client_type,omitempty"`
	MQTTClient string        `json:"client_id,omitempty" msg:"client_id,omitempty"`
	Nonce      string        `json:"nonce,omitempty" msg:"nonce,omitempty"`
}

// ForAssignmentSnap returns the minimal ClientInfo view that NATS
// uses when capturing assignment snapshots. We keep this here so
// our benchmark can closely mirror the server's behaviour.
func (ci *ClientInfo) ForAssignmentSnap() *ClientInfo {
	if ci == nil {
		return nil
	}
	return &ClientInfo{
		Account: ci.Account,
		Service: ci.Service,
		Cluster: ci.Cluster,
	}
}

// RaftGroup models the placement information for streams and
// consumers in the JetStream meta-layer.
type RaftGroup struct {
	Name      string      `json:"name" msg:"name"`
	Peers     []string    `json:"peers" msg:"peers"`
	Storage   StorageType `json:"store" msg:"store"`
	Cluster   string      `json:"cluster,omitempty" msg:"cluster,omitempty"`
	Preferred string      `json:"preferred,omitempty" msg:"preferred,omitempty"`
	ScaleUp   bool        `json:"scale_up,omitempty" msg:"scale_up,omitempty"`
}

// SequencePair tracks both stream and consumer sequence numbers for
// a given message, mirroring NATS' SequencePair.
type SequencePair struct {
	Consumer uint64 `json:"consumer_seq" msg:"consumer_seq"`
	Stream   uint64 `json:"stream_seq" msg:"stream_seq"`
}

// Pending represents a pending message for explicit/ack-all
// consumers. Only the fields relevant to JSON/CBOR are kept.
type Pending struct {
	Sequence  uint64 `json:"sequence" msg:"sequence"`
	Timestamp int64  `json:"ts" msg:"ts"`
}

// ConsumerState mirrors the NATS ConsumerState type sufficiently to
// exercise a realistic nested map workload when encoding.
type ConsumerState struct {
	Delivered   SequencePair         `json:"delivered" msg:"delivered"`
	AckFloor    SequencePair         `json:"ack_floor" msg:"ack_floor"`
	Pending     map[uint64]*Pending  `json:"pending,omitempty" msg:"pending,omitempty"`
	Redelivered map[uint64]uint64    `json:"redelivered,omitempty" msg:"redelivered,omitempty"`
}

// consumerAssignment mirrors just the subset of NATS' consumer
// assignment struct that participates in meta snapshots.
type consumerAssignment struct {
	Client     *ClientInfo    `json:"client,omitempty" msg:"client,omitempty"`
	Created    time.Time      `json:"created" msg:"created"`
	Name       string         `json:"name" msg:"name"`
	Stream     string         `json:"stream" msg:"stream"`
	ConfigJSON json.RawMessage `json:"consumer" msg:"consumer"`
	Group      *RaftGroup     `json:"group" msg:"group"`
	State      *ConsumerState `json:"state,omitempty" msg:"state,omitempty"`
	// Internal (not marshalled)
	pending bool `json:"-" msg:"-"`
}

// streamAssignment mirrors the NATS streamAssignment type, again
// limited to the fields that flow into writeable snapshots.
type streamAssignment struct {
	Client     *ClientInfo `json:"client,omitempty" msg:"client,omitempty"`
	Created    time.Time   `json:"created" msg:"created"`
	ConfigJSON json.RawMessage `json:"stream" msg:"stream"`
	Group      *RaftGroup  `json:"group" msg:"group"`
	Sync       string      `json:"sync" msg:"sync"`
	// Internal (not marshalled)
	consumers map[string]*consumerAssignment `json:"-" msg:"-"`
}

// WriteableConsumerAssignment is the on-the-wire consumer snapshot
// representation used by the JetStream meta snapshot.
type WriteableConsumerAssignment struct {
	Client     *ClientInfo    `json:"client,omitempty" msg:"client,omitempty"`
	Created    time.Time      `json:"created" msg:"created"`
	Name       string         `json:"name" msg:"name"`
	Stream     string         `json:"stream" msg:"stream"`
	ConfigJSON json.RawMessage `json:"consumer" msg:"consumer"`
	Group      *RaftGroup     `json:"group" msg:"group"`
	State      *ConsumerState `json:"state,omitempty" msg:"state,omitempty"`
}

// WriteableStreamAssignment is the on-the-wire stream snapshot
// representation used by the JetStream meta snapshot.
type WriteableStreamAssignment struct {
	Client     *ClientInfo                   `json:"client,omitempty" msg:"client,omitempty"`
	Created    time.Time                     `json:"created" msg:"created"`
	ConfigJSON json.RawMessage               `json:"stream" msg:"stream"`
	Group      *RaftGroup                    `json:"group" msg:"group"`
	Sync       string                        `json:"sync" msg:"sync"`
	Consumers  []*WriteableConsumerAssignment `json:"consumers,omitempty" msg:"consumers,omitempty"`
}

// MetaSnapshot is a simple wrapper type that holds the full set of
// writeable stream assignments. Defining this as a named type allows
// cborgen to generate MarshalCBOR/Decode* entrypoints for the whole
// snapshot in one call.
type MetaSnapshot struct {
	Streams []WriteableStreamAssignment `json:"streams" msg:"streams"`
}

// StreamConfigSnapshot and ConsumerConfigSnapshot are minimal
// configuration shapes used to generate realistic JSON blobs that are
// stored inside ConfigJSON fields.
type StreamConfigSnapshot struct {
	Name     string            `json:"name" msg:"name"`
	Subjects []string          `json:"subjects" msg:"subjects"`
	Storage  StorageType       `json:"storage" msg:"storage"`
	Metadata map[string]string `json:"metadata,omitempty" msg:"metadata,omitempty"`
}

type ConsumerConfigSnapshot struct {
	Durable       string            `json:"durable" msg:"durable"`
	MemoryStorage bool              `json:"mem_storage" msg:"mem_storage"`
	Metadata      map[string]string `json:"metadata,omitempty" msg:"metadata,omitempty"`
}
