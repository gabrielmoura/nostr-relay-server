package relaysync

import (
	"encoding/binary"
	"os"
	"testing"
)

func benchmarkMetadata(size int) []EventMetadata {
	items := make([]EventMetadata, size)
	for i := range items {
		binary.BigEndian.PutUint64(items[i].ID[24:], uint64(i))
		items[i].CreatedAt = int64(i / 1000)
	}
	return items
}

func BenchmarkRelaySyncDigest100K(b *testing.B) { benchmarkDigest(b, 100_000) }
func BenchmarkRelaySyncDigest1M(b *testing.B)   { benchmarkDigest(b, 1_000_000) }
func BenchmarkRelaySyncDigest10M(b *testing.B) {
	if os.Getenv("RELAYSYNC_BENCH_LARGE") == "" {
		b.Skip("set RELAYSYNC_BENCH_LARGE=1")
	}
	benchmarkDigest(b, 10_000_000)
}

func benchmarkDigest(b *testing.B, size int) {
	items := benchmarkMetadata(size)
	b.ReportMetric(float64(size), "events")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DigestMetadata(items); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRelaySyncZstd100K(b *testing.B) {
	codec, err := NewZstdCodec(ZstdOptions{Level: 1, Threshold: 1, MaxOutput: 16 << 20})
	if err != nil {
		b.Fatal(err)
	}
	defer codec.Close()
	payload := make([]byte, 100_000*96)
	for i := range payload {
		payload[i] = byte(i)
	}
	b.SetBytes(int64(len(payload)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := codec.Encode(payload); err != nil {
			b.Fatal(err)
		}
	}
}
