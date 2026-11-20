package lz4codec_test

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"testing"

	"github.com/pierrec/lz4/v4"

	"github.com/skycoin/skycoin-cx/t0-lz4codec"
)

func newCodec(t *testing.T, cfg lz4codec.Config) *lz4codec.Codec {
	t.Helper()
	c := &lz4codec.Codec{}
	if err := c.Init(cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil && !errors.Is(err, lz4codec.ErrClosed) {
			t.Error(err)
		}
	})
	return c
}

func goldenInputs() map[string][]byte {
	random := make([]byte, 4096)
	rand.New(rand.NewSource(1)).Read(random)
	return map[string][]byte{
		"short":  []byte("abc"),
		"random": random,
		"zeros":  make([]byte, 1<<20),
		"text":   bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 2000),
	}
}

// testdata/*.lz4 is the output of CompressBytesLZ4 before the codec existed.
func TestGolden(t *testing.T) {
	c := newCodec(t, lz4codec.Config{MaxInput: 1 << 20, MaxOutput: 1 << 20})
	// Each name runs twice, so a stale hash table would show up.
	for range 2 {
		for name, in := range goldenInputs() {
			want, err := os.ReadFile("testdata/" + name + ".lz4")
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.Compress(in)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s: compressed bytes differ from the old output", name)
			}
			out, err := c.Decompress(want)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !bytes.Equal(out, in) {
				t.Fatalf("%s: round trip changed the data", name)
			}
		}
	}
}

// Input that LZ4 cannot shrink used to panic. See #80.
func TestIncompressible(t *testing.T) {
	c := newCodec(t, lz4codec.Config{MaxInput: 4096, MaxOutput: 4096})
	for name, data := range map[string][]byte{"empty": {}, "short": []byte("abc"), "random": goldenInputs()["random"]} {
		b, err := c.Compress(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out, err := c.Decompress(b)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("%s: round trip: %v", name, err)
		}
	}
}

// The output is a plain LZ4 block that any decoder reads. See #80.
func TestHighRatio(t *testing.T) {
	data := make([]byte, 1<<20)
	c := newCodec(t, lz4codec.Config{MaxInput: len(data), MaxOutput: len(data)})
	b, err := c.Compress(data)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]byte, len(data))
	if n, err := lz4.UncompressBlock(b, out); err != nil || n != len(data) {
		t.Fatalf("lz4.UncompressBlock: n %d, err %v", n, err)
	}
}

// Buffers come from Init, so neither direction allocates.
func TestNoAllocations(t *testing.T) {
	data := goldenInputs()["text"]
	c := newCodec(t, lz4codec.Config{MaxInput: len(data), MaxOutput: len(data)})
	b, err := c.Compress(data)
	if err != nil {
		t.Fatal(err)
	}
	b = append([]byte(nil), b...)
	if n := testing.AllocsPerRun(10, func() { _, _ = c.Compress(data) }); n != 0 {
		t.Errorf("Compress allocates %v times", n)
	}
	if n := testing.AllocsPerRun(10, func() { _, _ = c.Decompress(b) }); n != 0 {
		t.Errorf("Decompress allocates %v times", n)
	}
}

func TestLimits(t *testing.T) {
	c := newCodec(t, lz4codec.Config{MaxInput: 16, MaxOutput: 1 << 10})
	if _, err := c.Compress(make([]byte, 17)); err == nil {
		t.Error("want an error for input over MaxInput")
	}
	big := newCodec(t, lz4codec.Config{MaxInput: 1 << 12, MaxOutput: 1 << 12})
	b, err := big.Compress(make([]byte, 1<<12))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Decompress(b); err == nil {
		t.Error("want an error for output over MaxOutput")
	}
	if _, err := c.Decompress([]byte{0xff, 0xff}); err == nil {
		t.Error("want an error for a corrupt block")
	}
}

func TestInitConfig(t *testing.T) {
	for _, cfg := range []lz4codec.Config{{}, {MaxInput: 1}, {MaxOutput: 1}, {MaxInput: -1, MaxOutput: 1}} {
		c := &lz4codec.Codec{}
		if err := c.Init(cfg); err == nil {
			t.Errorf("Init(%+v): want an error", cfg)
		}
		if _, err := c.Compress(nil); !errors.Is(err, lz4codec.ErrNotInitialized) {
			t.Errorf("after a failed Init: got %v, want ErrNotInitialized", err)
		}
	}
	c := newCodec(t, lz4codec.Config{MaxInput: 1, MaxOutput: 2})
	if got, err := c.Config(); err != nil || got != (lz4codec.Config{MaxInput: 1, MaxOutput: 2}) {
		t.Errorf("Config: %+v, %v", got, err)
	}
}

func TestMisuse(t *testing.T) {
	cfg := lz4codec.Config{MaxInput: 64, MaxOutput: 64}
	check := func(t *testing.T, c *lz4codec.Codec, want error) {
		t.Helper()
		if _, err := c.Compress([]byte("abc")); !errors.Is(err, want) {
			t.Errorf("Compress: got %v, want %v", err, want)
		}
		if _, err := c.Decompress([]byte{0x30, 'a', 'b', 'c'}); !errors.Is(err, want) {
			t.Errorf("Decompress: got %v, want %v", err, want)
		}
		if _, err := c.Config(); !errors.Is(err, want) {
			t.Errorf("Config: got %v, want %v", err, want)
		}
		if err := c.Close(); !errors.Is(err, want) {
			t.Errorf("Close: got %v, want %v", err, want)
		}
	}

	t.Run("before Init", func(t *testing.T) {
		check(t, &lz4codec.Codec{}, lz4codec.ErrNotInitialized)
	})
	t.Run("nil", func(t *testing.T) {
		var c *lz4codec.Codec
		check(t, c, lz4codec.ErrNilCodec)
		if err := c.Init(cfg); !errors.Is(err, lz4codec.ErrNilCodec) {
			t.Errorf("Init: got %v", err)
		}
	})
	t.Run("Init twice", func(t *testing.T) {
		c := newCodec(t, cfg)
		if err := c.Init(cfg); !errors.Is(err, lz4codec.ErrInitialized) {
			t.Errorf("got %v, want ErrInitialized", err)
		}
		if _, err := c.Compress([]byte("abc")); err != nil {
			t.Errorf("the second Init broke the codec: %v", err)
		}
	})
	t.Run("after Close", func(t *testing.T) {
		c := &lz4codec.Codec{}
		if err := c.Init(cfg); err != nil {
			t.Fatal(err)
		}
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		check(t, c, lz4codec.ErrClosed)
		if err := c.Init(cfg); !errors.Is(err, lz4codec.ErrClosed) {
			t.Errorf("Init after Close: got %v, want ErrClosed", err)
		}
	})
}
