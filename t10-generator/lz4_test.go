package generator_test

import (
	"bytes"
	"math/rand"
	"runtime"
	"testing"

	"github.com/pierrec/lz4"

	cxast "github.com/skycoin/skycoin-cx/t3-ast"
)

// Input that LZ4 cannot shrink used to panic with "data not compressible".
// See #80.
func TestCompression_LZ4Incompressible(t *testing.T) {
	random := make([]byte, 4096)
	rand.New(rand.NewSource(1)).Read(random)

	for name, data := range map[string][]byte{
		"empty":  {},
		"short":  []byte("abc"),
		"random": random,
	} {
		t.Run(name, func(t *testing.T) {
			b := append([]byte(nil), data...)
			cxast.CompressBytesLZ4(&b)
			cxast.UncompressBytesLZ4(&b)
			if !bytes.Equal(b, data) {
				t.Fatalf("round trip changed the data: got %d bytes, want %d", len(b), len(data))
			}
		})
	}
}

// Decompression used to allocate 1000 times the compressed size. See #80.
func TestCompression_LZ4UncompressAllocation(t *testing.T) {
	data := make([]byte, 1<<20)
	rand.New(rand.NewSource(2)).Read(data)
	b := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	b = b[:n]

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cxast.UncompressBytesLZ4(&b)
	runtime.ReadMemStats(&after)

	if !bytes.Equal(b, data) {
		t.Fatal("round trip changed the data")
	}
	if got := after.TotalAlloc - before.TotalAlloc; got > 16*uint64(len(data)) {
		t.Fatalf("allocated %d bytes to decompress %d", got, len(data))
	}
}

// Compressed bytes are a valid LZ4 block for any decoder. See #80.
func TestCompression_LZ4HighRatio(t *testing.T) {
	data := make([]byte, 1<<20)
	b := append([]byte(nil), data...)
	cxast.CompressBytesLZ4(&b)
	out := make([]byte, len(data))
	n, err := lz4.UncompressBlock(b, out)
	if err != nil || n != len(data) {
		t.Fatalf("lz4.UncompressBlock: n %d, err %v", n, err)
	}
	cxast.UncompressBytesLZ4(&b)
	if !bytes.Equal(b, data) {
		t.Fatal("round trip changed the data")
	}
}

// SerializeDebugInfo used to read the section offsets out of the LZ4 output.
// See #80.
func TestSerializeDebugInfoCompressed(t *testing.T) {
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	plain := cxast.SerializeCXProgram(prgrm, false, true)
	cxast.UncompressBytesLZ4(&plain)

	info := cxast.SerializeDebugInfo(prgrm, false, true)
	sum := info.Program + info.Calls + info.Packages + info.Structs + info.Functions +
		info.Expressions + info.Arguments + info.Integers + info.Strings + info.Memory
	if sum == 0 || sum > len(plain) {
		t.Fatalf("section sizes add up to %d, serialized program is %d bytes", sum, len(plain))
	}
}

// Deserialize ignored the decoder error. See #80.
func TestDeserializeCompressed(t *testing.T) {
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	b := cxast.SerializeCXProgram(prgrm, false, true)
	got := cxast.Deserialize(b, true)
	if len(got.Packages) != len(prgrm.Packages) {
		t.Fatalf("got %d packages, want %d", len(got.Packages), len(prgrm.Packages))
	}

	defer func() {
		if recover() == nil {
			t.Fatal("want a panic for truncated input")
		}
	}()
	short := cxast.SerializeCXProgram(prgrm, false, false)
	cxast.Deserialize(short[:8], false)
}
