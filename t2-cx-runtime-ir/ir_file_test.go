package ir

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-types"
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
		ImageLocals:      []Local{{LocalOffset: 0, LocalSize: 4, LocalName: 0, LocalTypeName: 1, LocalType: types.I32}, {LocalOffset: 4, LocalSize: 8, LocalType: types.UNDEFINED}},
		ImageLocalNames:  []string{"x", "i32", ""},
		ImageReadOnly:    []Span{{SpanStart: 4096, SpanEnd: 4098}},
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
	if again := ImageFileEncode(&decoded); string(again) != string(ImageFileEncode(&image)) || decoded.ImageEntryMain != 1 || decoded.ImageFuncs[1].FuncCode[0] != image.ImageFuncs[1].FuncCode[0] || decoded.ImageLocals[0].LocalType != types.I32 {
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
