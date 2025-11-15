package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/delaneyj/cbor/tests/jetstreammeta"
)

type benchResult struct {
	Name          string
	Size          int
	NsPerOp       float64
	MBPerSec      float64
	AllocsPerOp   float64
	MemBytesPerOp float64
	Err           error
}

func main() {
	streams := flag.Int("streams", jetstreammeta.DefaultNumStreams, "number of streams")
	consumers := flag.Int("consumers", jetstreammeta.DefaultNumConsumers, "number of consumers per stream")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "Building JetStream meta snapshot fixture (streams=%d, consumers=%d) ...\n", *streams, *consumers)
	snap := jetstreammeta.BuildMetaSnapshotFixture(*streams, *consumers)

	// Warm up encodings once to determine output sizes and surface any
	// obvious issues before running timed benchmarks.
	cborBuf, cborErr := snap.MarshalCBOR(nil)
	jsonBuf, jsonErr := json.Marshal(snap)
	msgSnap := jetstreammeta.ToMsgpMetaSnapshot(snap)
	msgpBuf, msgpErr := msgSnap.MarshalMsg(nil)

	rows := make([]benchResult, 0, 3)

	// Helper to compute MB/s from size and ns/op.
	mbps := func(size int, nsPerOp float64) float64 {
		if nsPerOp <= 0 {
			return 0
		}
		bytesPerSec := float64(size) * (1e9 / nsPerOp)
		return bytesPerSec / (1024 * 1024)
	}

	// CBOR (this library, using generated MetaSnapshot.MarshalCBOR).
	rows = append(rows, runCodecBench("CBOR (cbor/runtime)", len(cborBuf), cborErr, func(b *testing.B) {
		if cborErr != nil {
			return
		}
		buf := make([]byte, 0, len(cborBuf))
		b.SetBytes(int64(len(cborBuf)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var err error
			buf, err = snap.MarshalCBOR(buf[:0])
			if err != nil {
				b.Fatalf("MarshalCBOR: %v", err)
			}
		}
	}))

	// JSON baseline using encoding/json on the same MetaSnapshot
	// struct that cborgen and msgp target.
	rows = append(rows, runCodecBench("JSON (encoding/json)", len(jsonBuf), jsonErr, func(b *testing.B) {
		if jsonErr != nil {
			return
		}
		b.SetBytes(int64(len(jsonBuf)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := json.Marshal(snap); err != nil {
				b.Fatalf("json.Marshal: %v", err)
			}
		}
	}))

		// MessagePack baseline using tinylib/msgp's generated MarshalMsg
		// for a MsgpMetaSnapshot converted from the same MetaSnapshot
		// used by CBOR and JSON.
		rows = append(rows, runCodecBench("MSGP (generated MarshalMsg)", len(msgpBuf), msgpErr, func(b *testing.B) {
		if msgpErr != nil {
			return
		}
		buf := make([]byte, 0, len(msgpBuf))
		b.SetBytes(int64(len(msgpBuf)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var err error
			buf, err = msgSnap.MarshalMsg(buf[:0])
			if err != nil {
				b.Fatalf("MarshalMsg: %v", err)
			}
		}
	}))

	// Post-process rows to compute human-friendly metrics.
	for i := range rows {
		if rows[i].Err != nil || rows[i].Size == 0 || rows[i].NsPerOp <= 0 {
			continue
		}
		rows[i].MBPerSec = mbps(rows[i].Size, rows[i].NsPerOp)
	}

	printTable(rows, *streams, *consumers)
}

// runCodecBench runs a single benchmark function (if err is nil) and
// returns a populated benchResult. If err is non-nil, the benchmark
// is skipped and the error recorded.
func runCodecBench(name string, size int, err error, fn func(b *testing.B)) benchResult {
	res := benchResult{Name: name, Size: size, Err: err}
	if err != nil || size == 0 {
		return res
	}
	br := testing.Benchmark(fn)
	// testing.Benchmark guarantees br.N > 0 when it returns.
	res.NsPerOp = float64(br.NsPerOp())
	res.AllocsPerOp = float64(br.AllocsPerOp())
	res.MemBytesPerOp = float64(br.MemBytes) / float64(br.N)
	return res
}

// buildSnapshotView constructs a JSON/msgp-friendly view of the
// MetaSnapshot using only primitive types, maps, and slices. It
// mirrors the shape of the NATS meta snapshot JSON but normalizes
// time and duration fields into strings/integers so that both
// json/v2 and msgp.AppendIntf can serialize them.
func buildSnapshotView(snap jetstreammeta.MetaSnapshot) map[string]any {
	streams := make([]any, 0, len(snap.Streams))
	for i := range snap.Streams {
		streams = append(streams, buildStreamView(&snap.Streams[i]))
	}
	return map[string]any{
		"streams": streams,
	}
}

func buildStreamView(s *jetstreammeta.WriteableStreamAssignment) map[string]any {
	m := map[string]any{
		"created": s.Created.Format(time.RFC3339Nano),
		"stream":  s.ConfigJSON, // raw JSON bytes
		"sync":    s.Sync,
	}
	if s.Client != nil {
		m["client"] = buildClientView(s.Client)
	}
	if s.Group != nil {
		m["group"] = buildGroupView(s.Group)
	}
	if len(s.Consumers) > 0 {
		cons := make([]any, 0, len(s.Consumers))
		for _, ca := range s.Consumers {
			cons = append(cons, buildConsumerView(ca))
		}
		m["consumers"] = cons
	}
	return m
}

func buildClientView(ci *jetstreammeta.ClientInfo) map[string]any {
	m := map[string]any{}
	if ci.Account != "" {
		m["acc"] = ci.Account
	}
	if ci.Service != "" {
		m["svc"] = ci.Service
	}
	if ci.Cluster != "" {
		m["cluster"] = ci.Cluster
	}
	// RTT is a time.Duration; encode as nanoseconds for portability.
	if ci.RTT != 0 {
		m["rtt"] = int64(ci.RTT)
	}
	return m
}

func buildGroupView(rg *jetstreammeta.RaftGroup) map[string]any {
	m := map[string]any{
		"name":  rg.Name,
		"peers": append([]string(nil), rg.Peers...),
		"store": int(rg.Storage),
	}
	if rg.Cluster != "" {
		m["cluster"] = rg.Cluster
	}
	if rg.Preferred != "" {
		m["preferred"] = rg.Preferred
	}
	if rg.ScaleUp {
		m["scale_up"] = rg.ScaleUp
	}
	return m
}

func buildConsumerView(ca *jetstreammeta.WriteableConsumerAssignment) map[string]any {
	m := map[string]any{
		"created": ca.Created.Format(time.RFC3339Nano),
		"name":    ca.Name,
		"stream":  ca.Stream,
		"consumer": ca.ConfigJSON,
	}
	if ca.Client != nil {
		m["client"] = buildClientView(ca.Client)
	}
	if ca.Group != nil {
		m["group"] = buildGroupView(ca.Group)
	}
	if ca.State != nil {
		m["state"] = buildConsumerStateView(ca.State)
	}
	return m
}

func buildConsumerStateView(cs *jetstreammeta.ConsumerState) map[string]any {
	m := map[string]any{
		"delivered": buildSequencePairView(cs.Delivered),
		"ack_floor": buildSequencePairView(cs.AckFloor),
	}
		if len(cs.Pending) > 0 {
			pm := make(map[string]any, len(cs.Pending))
			for k, v := range cs.Pending {
				pm[fmt.Sprintf("%d", k)] = map[string]any{
				"sequence": v.Sequence,
				"ts":       v.Timestamp,
			}
		}
		m["pending"] = pm
	}
		if len(cs.Redelivered) > 0 {
			rm := make(map[string]any, len(cs.Redelivered))
			for k, v := range cs.Redelivered {
				rm[fmt.Sprintf("%d", k)] = v
		}
		m["redelivered"] = rm
	}
	return m
}

func buildSequencePairView(sp jetstreammeta.SequencePair) map[string]any {
	return map[string]any{
		"consumer_seq": sp.Consumer,
		"stream_seq":   sp.Stream,
	}
}

func printTable(rows []benchResult, streams, consumers int) {
	tw := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "# JetStream Meta Snapshot Encode Benchmarks (streams=%d, consumers=%d)\n", streams, consumers)
	fmt.Fprintf(tw, "# Timestamp: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(tw, "Codec\tBytes/op\tMB/s\tns/op\tAllocs/op\tMem/op (B)\tError")

	for _, r := range rows {
		if r.Err != nil {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t%v\n", r.Name, r.Size, r.Err)
			continue
		}
		if r.Size == 0 || r.NsPerOp <= 0 {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t(no data)\n", r.Name, r.Size)
			continue
		}
		fmt.Fprintf(tw, "%s\t%d\t%.2f\t%.0f\t%.2f\t%.0f\t-\n",
			r.Name,
			r.Size,
			r.MBPerSec,
			r.NsPerOp,
			r.AllocsPerOp,
			r.MemBytesPerOp,
		)
	}

	_ = tw.Flush()
}
