//go:build cxfx

package cxfx

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
)

// decodeTestFile writes data to a file in a temporary directory and opens it.
func decodeTestFile(t *testing.T, data []byte) *os.File {
	path := filepath.Join(t.TempDir(), "image")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

func TestDecodeHdr(t *testing.T) {
	header := "#?RADIANCE\n# made by a test\nFORMAT=32-bit_rle_rgbe\n\n-Y 2 +X 3\n"
	pixels := []byte{128, 64, 32, 129, 0, 0, 0, 128, 255, 255, 255, 120, 1, 2, 3, 130, 64, 64, 64, 127, 200, 100, 50, 128}
	data, width, height := decodeHdr(decodeTestFile(t, append([]byte(header), pixels...)))
	if width != 3 || height != 2 || len(data) != 3*2*3*4 {
		t.Fatalf("width %d height %d with %d bytes", width, height, len(data))
	}
	want := []float32{1, 0.5, 0.25, 0, 0, 0}
	for i, value := range want {
		if got := types.Read_f32(data, types.Pointer(i*4)); got != value {
			t.Errorf("channel %d is %v, want %v", i, got, value)
		}
	}
	if got := types.Read_f32(data, 3*3*4); got != 4.0/256 {
		t.Errorf("the second row starts at %v", got)
	}
	for _, bad := range []string{
		"#?RADIANCE\n\n-Y 2 +X 3\n",
		"#?RADIANCE\nFORMAT=32-bit_rle_rgbe\n\n+Y 2 +X 3\n",
		"#?RADIANCE\nFORMAT=32-bit_rle_rgbe\n\n-Y 2 +X\n",
	} {
		if data, _, _ := decodeHdr(decodeTestFile(t, []byte(bad))); data != nil {
			t.Errorf("%q decoded", bad)
		}
	}
}

func TestDecodeImg(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 3, 2))
	draw.Draw(source, source.Bounds(), &image.Uniform{color.RGBA{200, 100, 50, 255}}, image.Point{}, draw.Src)
	source.Set(1, 1, color.RGBA{10, 20, 30, 255})
	encoders := []struct {
		name   string
		encode func(*bytes.Buffer) error
	}{
		{"png", func(b *bytes.Buffer) error { return png.Encode(b, source) }},
		{"gif", func(b *bytes.Buffer) error { return gif.Encode(b, source, nil) }},
		{"jpeg", func(b *bytes.Buffer) error { return jpeg.Encode(b, source, nil) }},
	}
	for _, encoder := range encoders {
		name, encode := encoder.name, encoder.encode
		var encoded bytes.Buffer
		if err := encode(&encoded); err != nil {
			t.Fatal(err)
		}
		decoded, _, err := image.Decode(bytes.NewReader(encoded.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		want := image.NewRGBA(decoded.Bounds())
		draw.Draw(want, want.Bounds(), decoded, image.Point{}, draw.Src)
		data, width, height, pixels := decodeImg(decodeTestFile(t, encoded.Bytes()), true)
		if width != 3 || height != 2 || !bytes.Equal(data, want.Pix) || len(pixels) != 3*2*4 {
			t.Errorf("%s: %dx%d, %d pixels, data %v, want %v", name, width, height, len(pixels), data, want.Pix)
		}
	}
	if _, err := imageDecode(decodeTestFile(t, []byte("BM not an image"))); err != image.ErrFormat {
		t.Errorf("an unknown format is %v", err)
	}
}
