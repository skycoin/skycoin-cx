package generator_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-lz4codec"
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
)

// newProgramCodec sizes a codec for the sample programs, whose whole memory
// is about 3 MiB.
func newProgramCodec(t *testing.T) *lz4codec.Codec {
	t.Helper()
	c := &lz4codec.Codec{}
	if err := c.Init(lz4codec.Config{MaxInput: 8 << 20, MaxOutput: 8 << 20}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	return c
}

// testdata/program.lz4 is SerializeCXProgram output from before the codec
// existed. Serialization has changed since, so the check is that
// the codec maps those bytes to the same plain bytes and back to the same
// compressed bytes.
func TestCompression_LZ4ProgramGolden(t *testing.T) {
	const sum = "5483f24fe2bdb73e2df1284734418950fb1410c3ddd104f8aa6b56c36d6f35bf"
	c := newProgramCodec(t)
	want, err := os.ReadFile("testdata/program.lz4")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := c.Decompress(want)
	if err != nil {
		t.Fatal(err)
	}
	if h := sha256.Sum256(plain); hex.EncodeToString(h[:]) != sum {
		t.Fatal("decompressed bytes differ from the old output")
	}
	plain = append([]byte(nil), plain...)
	got, err := c.Compress(plain)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("compressed bytes differ from the old output")
	}
}

// SerializeDebugInfo used to read the section offsets out of the LZ4 output.
// See #80.
func TestSerializeDebugInfoCompressed(t *testing.T) {
	c := newProgramCodec(t)
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	b, err := cxast.SerializeCXProgram(prgrm, false, c)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := c.Decompress(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) < len(prgrm.Memory) {
		t.Fatalf("%d plain bytes do not hold %d bytes of memory", len(plain), len(prgrm.Memory))
	}

	info, err := cxast.SerializeDebugInfo(prgrm, false, true)
	if err != nil {
		t.Fatal(err)
	}
	sum := info.Program + info.Calls + info.Packages + info.Structs + info.Functions +
		info.Expressions + info.Arguments + info.Integers + info.Strings + info.Memory
	if sum == 0 || sum > len(plain) {
		t.Fatalf("section sizes add up to %d, serialized program is %d bytes", sum, len(plain))
	}
}

// Deserialize ignored the decoder error. See #80.
func TestDeserializeCompressed(t *testing.T) {
	c := newProgramCodec(t)
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	b, err := cxast.SerializeCXProgram(prgrm, false, c)
	if err != nil {
		t.Fatal(err)
	}
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0016.
	got, err := cxast.Deserialize(b, c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != len(prgrm.Packages) {
		t.Fatalf("got %d packages, want %d", len(got.Packages), len(prgrm.Packages))
	}

	short, err := cxast.SerializeCXProgram(prgrm, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0017.
	if _, err := cxast.Deserialize(short[:8], nil); err == nil {
		t.Fatal("want an error for truncated input")
	}
}

// A codec that is not initialized or is closed is an error, not a panic.
func TestSerializeCodecMisuse(t *testing.T) {
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	closed := &lz4codec.Codec{}
	if err := closed.Init(lz4codec.Config{MaxInput: 1, MaxOutput: 1}); err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		c    *lz4codec.Codec
		want error
	}{
		"before Init": {&lz4codec.Codec{}, lz4codec.ErrNotInitialized},
		"after Close": {closed, lz4codec.ErrClosed},
	} {
		if _, err := cxast.SerializeCXProgram(prgrm, false, tc.c); !errors.Is(err, tc.want) {
			t.Errorf("%s: SerializeCXProgram: got %v, want %v", name, err, tc.want)
		}
		// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
		// Now a constructor call, site S0018.
		if _, err := cxast.Deserialize([]byte{0}, tc.c); !errors.Is(err, tc.want) {
			t.Errorf("%s: Deserialize: got %v, want %v", name, err, tc.want)
		}
	}
}

// A program serialized under another language version has different opcode
// numbers, so deserializing it must fail. See #183.
func TestDeserializeRejectsOtherVersion(t *testing.T) {
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	b, err := cxast.SerializeCXProgram(prgrm, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0019.
	got, err := cxast.Deserialize(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != prgrm.Version {
		t.Fatalf("got version %d, want %d", got.Version, prgrm.Version)
	}

	prgrm.Version = 1
	if b, err = cxast.SerializeCXProgram(prgrm, false, nil); err != nil {
		t.Fatal(err)
	}
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0020.
	if _, err := cxast.Deserialize(b, nil); err == nil {
		t.Fatal("want an error for a program from another version")
	}
}
