package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"
	"time"

	js "github.com/delaneyj/cbor/tests/jetstreammeta"
	msgpjs "github.com/delaneyj/cbor/benchmarks/jetstreammeta_msgp"
)

type benchResult struct {
	Name             string
	Size             int
	EncNsPerOp       float64
	EncMBPerSec      float64
	EncAllocsPerOp   float64
	EncMemBytesPerOp float64
	DecNsPerOp       float64
	DecMBPerSec      float64
	DecAllocsPerOp   float64
	DecMemBytesPerOp float64
	Err              error
}

func main() {
	streams := flag.Int("streams", js.DefaultNumStreams, "number of streams")
	consumers := flag.Int("consumers", js.DefaultNumConsumers, "number of consumers per stream")
	flag.Parse()

	fmt.Fprintf(os.Stderr, "Building JetStream meta snapshot fixture (streams=%d, consumers=%d) ...\n", *streams, *consumers)
	snap := js.BuildMetaSnapshotFixture(*streams, *consumers)

	cborBuf, cborErr := snap.MarshalCBOR(nil)
	jsonBuf, jsonErr := json.Marshal(snap)
	msgSnap := msgpjs.ToMsgpMetaSnapshot(snap)
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
				var dst js.MetaSnapshot
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
				var dst js.MetaSnapshot
				if err := json.Unmarshal(jsonBuf, &dst); err != nil {
					b.Fatalf("json.Unmarshal: %v", err)
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
				var dst msgpjs.MsgpMetaSnapshot
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
