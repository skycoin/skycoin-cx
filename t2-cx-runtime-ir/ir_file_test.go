package ir

import (
	"encoding/binary"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

func irFileTestImage() Image {
	image := Image{
		ImageFuncs: []Func{
			{FuncCode: []Instr{{InstrOpcode: OpcodeReturn}}},
			{
				FuncCode:      []Instr{{InstrOpcode: 3, InstrMode: InstrModeFrameA | InstrModeFrameC, InstrA: 1, InstrB: 2, InstrC: 3, InstrD: 4, InstrN: 5, InstrM: 6, InstrAux: 7}, {InstrOpcode: OpcodeReturn}},
				FuncFrameSize: 24,
				FuncInputs:    []Slot{{SlotOffset: 0, SlotSize: 4}},
				FuncOutputs:   []Slot{{SlotOffset: 4, SlotSize: 8}, {SlotOffset: 12, SlotSize: 4}},
				FuncLineFirst: 0, FuncLineCount: 1, FuncLocalFirst: 0, FuncLocalCount: 2,
			},
		},
		ImageData:        []byte{1, 2, 3, 4, 5},
		ImageDataStart:   4096,
		ImageStackSize:   4096,
		ImageScratchSize: 64,
		ImageText:        []byte("hello"),
		ImageEntryInit:   ImageFuncNone,
		ImageEntryMain:   1,
		ImageLines:       []Line{{LinePc: 0, LineFile: 0, LineNumber: 3}},
		ImageFileNames:   []string{"main.cx"},
		ImageFuncNames:   []string{"*init", "main"},
		ImageLocals:      []Local{{LocalOffset: 0, LocalSize: 4, LocalName: 0, LocalTypeName: 1, LocalType: valuetypes.I32}, {LocalOffset: 4, LocalSize: 8, LocalType: valuetypes.UNDEFINED}},
		ImageLocalNames:  []string{"x", "i32", ""},
		ImageReadOnly:    []Span{{SpanStart: 4096, SpanEnd: 4098}},
		ImageOutputMode:  output.ModeThread,
	}
	image.ImageOperationsEnabled[0] = 0x8000000000000001
	return image
}

func TestImageFileRoundTrip(t *testing.T) {
	image := irFileTestImage()
	decoded, problem := ImageFileDecode(ImageFileEncode(&image))
	if problem != ImageFileProblemNone {
		t.Fatalf("decode: %s", ImageFileProblemMessages[problem])
	}
	if again := ImageFileEncode(&decoded); string(again) != string(ImageFileEncode(&image)) || decoded.ImageEntryMain != 1 || decoded.ImageFuncs[1].FuncCode[0] != image.ImageFuncs[1].FuncCode[0] || decoded.ImageLocals[0].LocalType != valuetypes.I32 {
		t.Fatalf("decoded image differs:\n%+v\nwant\n%+v", decoded, image)
	}
}

func TestImageFileRefusals(t *testing.T) {
	image := irFileTestImage()
	file := ImageFileEncode(&image)
	if _, problem := ImageFileDecode([]byte("not an image")); problem != ImageFileProblemMagic {
		t.Errorf("other bytes: problem %d, want magic", problem)
	}
	for size := len(ImageFileMagic); size < len(file); size++ {
		if _, problem := ImageFileDecode(file[:size]); problem != ImageFileProblemTruncated {
			t.Fatalf("file cut to %d of %d bytes: problem %d, want truncated", size, len(file), problem)
		}
	}
	if _, problem := ImageFileDecode(append(append([]byte{}, file...), 0)); problem != ImageFileProblemTruncated {
		t.Errorf("a byte after the end: problem %d, want truncated", problem)
	}
	other := append([]byte{}, file...)
	other[len(ImageFileMagic)+4]++
	if _, problem := ImageFileDecode(other); problem != ImageFileProblemVersion {
		t.Errorf("another opcode count: problem %d, want version", problem)
	}
}

// imageFileEncodeAppend is the encoder ImageFileEncode replaced, kept as the
// oracle of ImageFileEncodeInto.
func imageFileEncodeAppend(image *Image) []byte {
	file := []byte(ImageFileMagic)
	file = binary.LittleEndian.AppendUint32(file, ImageFileVersion)
	file = binary.LittleEndian.AppendUint32(file, uint32(OpcodeCount))
	file = binary.LittleEndian.AppendUint32(file, InstrSize)
	file = binary.LittleEndian.AppendUint32(file, uint32(len(image.ImageFuncs)))
	for i := range image.ImageFuncs {
		fn := &image.ImageFuncs[i]
		file = binary.LittleEndian.AppendUint32(file, uint32(len(fn.FuncCode)))
		for _, in := range fn.FuncCode {
			file = binary.LittleEndian.AppendUint16(file, uint16(in.InstrOpcode))
			file = binary.LittleEndian.AppendUint16(file, uint16(in.InstrMode))
			for _, operand := range [...]uint32{in.InstrA, in.InstrB, in.InstrC, in.InstrD, in.InstrN, in.InstrM, in.InstrAux} {
				file = binary.LittleEndian.AppendUint32(file, operand)
			}
		}
		file = binary.LittleEndian.AppendUint32(file, fn.FuncFrameSize)
		file = imageFileSlotsAppend(file, fn.FuncInputs)
		file = imageFileSlotsAppend(file, fn.FuncOutputs)
		for _, value := range [...]uint32{fn.FuncLineFirst, fn.FuncLineCount, fn.FuncLocalFirst, fn.FuncLocalCount} {
			file = binary.LittleEndian.AppendUint32(file, value)
		}
	}
	file = imageFileBytesAppend(file, image.ImageData)
	for _, value := range [...]uint32{image.ImageDataStart, image.ImageStackSize, image.ImageScratchSize} {
		file = binary.LittleEndian.AppendUint32(file, value)
	}
	file = imageFileBytesAppend(file, image.ImageText)
	file = binary.LittleEndian.AppendUint32(file, image.ImageEntryInit)
	file = binary.LittleEndian.AppendUint32(file, image.ImageEntryMain)
	file = binary.LittleEndian.AppendUint32(file, uint32(len(image.ImageLines)))
	for _, line := range image.ImageLines {
		file = binary.LittleEndian.AppendUint32(file, line.LinePc)
		file = binary.LittleEndian.AppendUint32(file, line.LineFile)
		file = binary.LittleEndian.AppendUint32(file, line.LineNumber)
	}
	file = imageFileStringsAppend(file, image.ImageFileNames)
	file = imageFileStringsAppend(file, image.ImageFuncNames)
	file = binary.LittleEndian.AppendUint32(file, uint32(len(image.ImageLocals)))
	for _, local := range image.ImageLocals {
		file = binary.LittleEndian.AppendUint32(file, local.LocalOffset)
		file = binary.LittleEndian.AppendUint32(file, local.LocalSize)
		file = binary.LittleEndian.AppendUint32(file, local.LocalName)
		file = binary.LittleEndian.AppendUint32(file, local.LocalTypeName)
		file = binary.LittleEndian.AppendUint32(file, uint32(int32(local.LocalType)))
	}
	file = imageFileStringsAppend(file, image.ImageLocalNames)
	file = binary.LittleEndian.AppendUint32(file, uint32(len(image.ImageReadOnly)))
	for _, span := range image.ImageReadOnly {
		file = binary.LittleEndian.AppendUint32(file, span.SpanStart)
		file = binary.LittleEndian.AppendUint32(file, span.SpanEnd)
	}
	file = binary.LittleEndian.AppendUint32(file, opcodetable.OpcodeSetWordCount)
	for _, word := range image.ImageOperationsEnabled {
		file = binary.LittleEndian.AppendUint64(file, word)
	}
	file = binary.LittleEndian.AppendUint32(file, uint32(image.ImageOutputMode))
	return file
}

func imageFileSlotsAppend(file []byte, slots []Slot) []byte {
	file = binary.LittleEndian.AppendUint32(file, uint32(len(slots)))
	for _, slot := range slots {
		file = binary.LittleEndian.AppendUint32(file, slot.SlotOffset)
		file = binary.LittleEndian.AppendUint32(file, slot.SlotSize)
	}
	return file
}

func imageFileBytesAppend(file []byte, bytes []byte) []byte {
	file = binary.LittleEndian.AppendUint32(file, uint32(len(bytes)))
	return append(file, bytes...)
}

func imageFileStringsAppend(file []byte, strs []string) []byte {
	file = binary.LittleEndian.AppendUint32(file, uint32(len(strs)))
	for _, str := range strs {
		file = binary.LittleEndian.AppendUint32(file, uint32(len(str)))
		file = append(file, str...)
	}
	return file
}

// TestImageFileEncodeInto checks ImageFileEncodeInto byte for byte against
// the append encoder and that it allocates nothing.
func TestImageFileEncodeInto(t *testing.T) {
	image := irFileTestImage()
	want := imageFileEncodeAppend(&image)
	file := make([]byte, len(want)+3)
	if size := ImageFileEncodeInto(file, &image); size != len(want) || string(file[:size]) != string(want) {
		t.Fatalf("ImageFileEncodeInto wrote %d bytes that differ from the %d of the append encoder", size, len(want))
	}
	if size := ImageFileEncodeInto(file[:len(want)-1], &image); size != -1 {
		t.Errorf("a short file gives %d, want -1", size)
	}
	if allocs := testing.AllocsPerRun(10, func() { ImageFileEncodeInto(file, &image) }); allocs != 0 {
		t.Errorf("ImageFileEncodeInto allocates %.0f times", allocs)
	}
}
