package jetstreammeta

import (
	"encoding/json"
	"fmt"
	"time"
)

// Default fixture sizes chosen to mirror the NATS
// BenchmarkJetStreamMetaSnapshot benchmark: 200 streams with
// 500 consumers each.
const (
	DefaultNumStreams   = 200
	DefaultNumConsumers = 500
)

// BuildMetaSnapshotFixture constructs a MetaSnapshot value that closely
// resembles the structure and scale of the JetStream meta snapshot
// used in the NATS server benchmarks.
func BuildMetaSnapshotFixture(numStreams, numConsumers int) MetaSnapshot {
	if numStreams <= 0 {
		numStreams = DefaultNumStreams
	}
	if numConsumers <= 0 {
		numConsumers = DefaultNumConsumers
	}

	// Single logical account/cluster for the whole fixture.
	client := &ClientInfo{
		Account: "G",
		Service: "JS",
		Cluster: "R3S",
		Name:    "bench-meta",
	}

	rg := &RaftGroup{
		Name:    "rg-meta",
		Peers:   []string{"n1", "n2", "n3"},
		Storage: MemoryStorage,
		Cluster: "R3S",
	}

	metadata := map[string]string{
		"required_api": "0",
	}

	streamsByName := make(map[string]*streamAssignment, numStreams)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < numStreams; i++ {
		streamName := fmt.Sprintf("STREAM-%d", i)
		subject := fmt.Sprintf("SUBJECT-%d", i)

		cfg := StreamConfigSnapshot{
			Name:     streamName,
			Subjects: []string{subject},
			Storage:  MemoryStorage,
			Metadata: metadata,
		}
		cfgJSON, _ := json.Marshal(cfg)

			sa := &streamAssignment{
				Client:     client,
				Created:    baseTime.Add(time.Duration(i) * time.Millisecond),
				ConfigJSON: json.RawMessage(cfgJSON),
			Group:      rg,
			Sync:       "_INBOX.meta.sync",
			consumers:  make(map[string]*consumerAssignment, numConsumers),
		}

		for j := 0; j < numConsumers; j++ {
			consumerName := fmt.Sprintf("CONSUMER-%d", j)
			ccfg := ConsumerConfigSnapshot{
				Durable:       consumerName,
				MemoryStorage: true,
				Metadata:      metadata,
			}
			ccfgJSON, _ := json.Marshal(ccfg)

			state := &ConsumerState{
				Delivered: SequencePair{
					Consumer: uint64(j + 1),
					Stream:   uint64(j + 1),
				},
				AckFloor: SequencePair{
					Consumer: uint64(j),
					Stream:   uint64(j),
				},
				Pending: map[uint64]*Pending{
					1: {
						Sequence:  uint64(j + 1),
						Timestamp: baseTime.Add(time.Duration(i*j) * time.Millisecond).UnixNano(),
					},
				},
				Redelivered: map[uint64]uint64{
					1: 2,
				},
			}

			ca := &consumerAssignment{
				Client:     client,
				Created:    sa.Created,
				Name:       consumerName,
				Stream:     streamName,
				ConfigJSON: json.RawMessage(ccfgJSON),
				Group:      rg,
				State:      state,
			}

			sa.consumers[consumerName] = ca
		}

		streamsByName[streamName] = sa
	}

	// Now mirror js.metaSnapshot: transform streamAssignment and
	// consumerAssignment to their writeable forms.
	streams := make([]WriteableStreamAssignment, 0, len(streamsByName))
	for _, sa := range streamsByName {
		wsa := WriteableStreamAssignment{
			Client:     sa.Client.ForAssignmentSnap(),
			Created:    sa.Created,
			ConfigJSON: sa.ConfigJSON,
			Group:      sa.Group,
			Sync:       sa.Sync,
			Consumers:  make([]*WriteableConsumerAssignment, 0, len(sa.consumers)),
		}
		for _, ca := range sa.consumers {
			if ca.pending {
				continue
			}
			wca := WriteableConsumerAssignment{
				Client:     ca.Client.ForAssignmentSnap(),
				Created:    ca.Created,
				Name:       ca.Name,
				Stream:     ca.Stream,
				ConfigJSON: ca.ConfigJSON,
				Group:      ca.Group,
				State:      ca.State,
			}
			wsa.Consumers = append(wsa.Consumers, &wca)
		}
		streams = append(streams, wsa)
	}

	return MetaSnapshot{Streams: streams}
}
