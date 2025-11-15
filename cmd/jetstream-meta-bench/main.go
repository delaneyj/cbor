package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	jsonv2 "github.com/go-json-experiment/json"
	"github.com/delaneyj/cbor/tests/jetstreammeta"
)

type benchResult struct {
	Name          string
	Size          int
	EncNsPerOp       float64
	EncMBPerSec      float64
	EncAllocsPerOp   float64
	EncMemBytesPerOp float64
	DecNsPerOp       float64
	DecMBPerSec      float64
	DecAllocsPerOp   float64
	DecMemBytesPerOp float64
	Err           error
}

func main() {
	streams := flag.Int("streams", jetstreammeta.DefaultNumStreams, "number of streams")
	consumers := flag.Int("consumers", jetstreammeta.DefaultNumConsumers, "number of consumers per stream")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "Building JetStream meta snapshot fixture (streams=%d, consumers=%d) ...\n", *streams, *consumers)
	snap := jetstreammeta.BuildMetaSnapshotFixture(*streams, *consumers)
	view := buildSnapshotView(snap)

	cborBuf, cborErr := snap.MarshalCBOR(nil)
	jsonBuf, jsonErr := json.Marshal(snap)
	jsonV2Buf, jsonV2Err := jsonv2.Marshal(view)
	msgSnap := jetstreammeta.ToMsgpMetaSnapshot(snap)
	msgpBuf, msgpErr := msgSnap.MarshalMsg(nil)

	rows := make([]benchResult, 0, 4)

	rows = append(rows, runCodecBench("CBOR (cbor/runtime)", len(cborBuf), cborErr,
		func(b *testing.B) { // encode
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
		},
		func(b *testing.B) { // decode
			if cborErr != nil {
				return
			}
			b.SetBytes(int64(len(cborBuf)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var dst jetstreammeta.MetaSnapshot
				rest, err := dst.DecodeTrusted(cborBuf)
				if err != nil || len(rest) != 0 {
					b.Fatalf("DecodeTrusted: %v (rest=%d)", err, len(rest))
				}
			}
		},
	))

	rows = append(rows, runCodecBench("JSON (encoding/json)", len(jsonBuf), jsonErr,
		func(b *testing.B) { // encode
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
		},
		func(b *testing.B) { // decode
			if jsonErr != nil {
				return
			}
			b.SetBytes(int64(len(jsonBuf)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var dst jetstreammeta.MetaSnapshot
				if err := json.Unmarshal(jsonBuf, &dst); err != nil {
					b.Fatalf("json.Unmarshal: %v", err)
				}
			}
		},
	))

	rows = append(rows, runCodecBench("JSON v2 (json-experiment)", len(jsonV2Buf), jsonV2Err,
		func(b *testing.B) { // encode
			if jsonV2Err != nil {
				return
			}
			b.SetBytes(int64(len(jsonV2Buf)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := jsonv2.Marshal(view); err != nil {
					b.Fatalf("jsonv2.Marshal: %v", err)
				}
			}
		},
		func(b *testing.B) { // decode
			if jsonV2Err != nil {
				return
			}
			b.SetBytes(int64(len(jsonV2Buf)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var dst map[string]any
				if err := jsonv2.Unmarshal(jsonV2Buf, &dst); err != nil {
					b.Fatalf("jsonv2.Unmarshal: %v", err)
				}
			}
		},
	))

	rows = append(rows, runCodecBench("MSGP (generated MarshalMsg)", len(msgpBuf), msgpErr,
		func(b *testing.B) { // encode
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
		},
		func(b *testing.B) { // decode
			if msgpErr != nil {
				return
			}
			b.SetBytes(int64(len(msgpBuf)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var dst jetstreammeta.MsgpMetaSnapshot
				rest, err := dst.UnmarshalMsg(msgpBuf)
				if err != nil || len(rest) != 0 {
					b.Fatalf("UnmarshalMsg: %v (rest=%d)", err, len(rest))
				}
			}
		},
	))

	printTable(rows, *streams, *consumers)
}

func runCodecBench(name string, size int, err error, enc, dec func(b *testing.B)) benchResult {
	res := benchResult{Name: name, Size: size, Err: err}
	if err != nil || size == 0 {
		return res
	}

	mbps := func(size int, nsPerOp float64) float64 {
		if nsPerOp <= 0 {
			return 0
		}
		bytesPerSec := float64(size) * (1e9 / nsPerOp)
		return bytesPerSec / (1024 * 1024)
	}

	if enc != nil {
		br := testing.Benchmark(enc)
		res.EncNsPerOp = float64(br.NsPerOp())
		res.EncAllocsPerOp = float64(br.AllocsPerOp())
		res.EncMemBytesPerOp = float64(br.MemBytes) / float64(br.N)
		res.EncMBPerSec = mbps(size, res.EncNsPerOp)
	}
	if dec != nil {
		br := testing.Benchmark(dec)
		res.DecNsPerOp = float64(br.NsPerOp())
		res.DecAllocsPerOp = float64(br.AllocsPerOp())
		res.DecMemBytesPerOp = float64(br.MemBytes) / float64(br.N)
		res.DecMBPerSec = mbps(size, res.DecNsPerOp)
	}
	return res
}

func buildSnapshotView(snap jetstreammeta.MetaSnapshot) map[string]any {
	streams := make([]any, 0, len(snap.Streams))
	for i := range snap.Streams {
		streams = append(streams, buildStreamView(&snap.Streams[i]))
	}
	return map[string]any{"streams": streams}
}

func buildStreamView(s *jetstreammeta.WriteableStreamAssignment) map[string]any {
	m := map[string]any{
		"created": s.Created.Format(time.RFC3339Nano),
		"stream":  s.ConfigJSON,
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
		"created":  ca.Created.Format(time.RFC3339Nano),
		"name":     ca.Name,
		"stream":   ca.Stream,
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
	fmt.Fprintln(tw, "Codec\tBytes/op\tEnc MB/s\tEnc ns/op\tEnc Allocs/op\tEnc Mem/op (B)\tError")
	for _, r := range rows {
		if r.Err != nil {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t%v\n", r.Name, r.Size, r.Err)
			continue
		}
		if r.Size == 0 || r.EncNsPerOp <= 0 {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t(no data)\n", r.Name, r.Size)
			continue
		}
		fmt.Fprintf(tw, "%s\t%d\t%.2f\t%.0f\t%.2f\t%.0f\t-\n", r.Name, r.Size, r.EncMBPerSec, r.EncNsPerOp, r.EncAllocsPerOp, r.EncMemBytesPerOp)
	}
	_ = tw.Flush()

	fmt.Println()

	tw = tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "# JetStream Meta Snapshot Decode Benchmarks (streams=%d, consumers=%d)\n", streams, consumers)
	fmt.Fprintf(tw, "# Timestamp: %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(tw, "Codec\tBytes/op\tDec MB/s\tDec ns/op\tDec Allocs/op\tDec Mem/op (B)\tError")
	for _, r := range rows {
		if r.Err != nil {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t%v\n", r.Name, r.Size, r.Err)
			continue
		}
		if r.Size == 0 || r.DecNsPerOp <= 0 {
			fmt.Fprintf(tw, "%s\t%d\t-\t-\t-\t-\t(no data)\n", r.Name, r.Size)
			continue
		}
		fmt.Fprintf(tw, "%s\t%d\t%.2f\t%.0f\t%.2f\t%.0f\t-\n", r.Name, r.Size, r.DecMBPerSec, r.DecNsPerOp, r.DecAllocsPerOp, r.DecMemBytesPerOp)
	}
	_ = tw.Flush()
}
