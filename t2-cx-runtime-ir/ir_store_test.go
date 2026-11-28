package ir

import (
	"testing"
)

// irStoreTestRoom is a small room, so the tests reach its bounds.
var irStoreTestRoom = ImageStoreRoom{
	ImageStoreRoomImages: 2, ImageStoreRoomFunctions: 4, ImageStoreRoomInstructions: 8, ImageStoreRoomSlots: 8,
	ImageStoreRoomLines: 4, ImageStoreRoomLocals: 4, ImageStoreRoomSpans: 4, ImageStoreRoomFileNames: 4,
	ImageStoreRoomLocalNames: 4, ImageStoreRoomSegmentBytes: 64, ImageStoreRoomTextBytes: 16, ImageStoreRoomNames: 1024, ImageStoreRoomNameBytes: 16 << 10,
}

// TestImageStoreFileRoundTrip decodes an image file into a store and checks
// that it encodes to the same bytes, and that a damaged file takes no slot.
func TestImageStoreFileRoundTrip(t *testing.T) {
	image := irFileTestImage()
	file := ImageFileEncode(&image)
	store := ImageStoreNew(irStoreTestRoom)
	for size := 0; size < len(file); size++ {
		if imageGid := store.ImageStoreFileDecode(file[:size]); imageGid != -1 {
			t.Fatalf("a file cut to %d of %d bytes decodes to %#x", size, len(file), imageGid)
		}
	}
	imageGid := store.ImageStoreFileDecode(file)
	if imageGid <= 0 {
		t.Fatalf("decode gives %d", imageGid)
	}
	again := make([]byte, len(file))
	if size := store.ImageStoreFileEncode(nil, imageGid, again); size != int32(len(file)) || string(again) != string(file) {
		t.Fatalf("encode of the decoded image gives %d bytes that differ", size)
	}
	if store.ImageStoreFunctionNameGet(nil, imageGid, 1) != store.ImageStoreStringPoolTextAppend([]byte("main")) {
		t.Errorf("function 1 is not named main")
	}
	if store.ImageStoreImageCreate() <= 0 || store.ImageStoreImageCreate() != 0 || store.ImageStoreFileDecode(file) != -1 {
		t.Errorf("a store of two images takes a third")
	}
}

// TestImageStoreWrites builds an image and checks that the running image
// and an image that runs refuse every write.
func TestImageStoreWrites(t *testing.T) {
	store := ImageStoreNew(irStoreTestRoom)
	current := irFileTestImage()
	imageGid := store.ImageStoreImageCreate()
	name := store.ImageStoreStringPoolTextAppend([]byte("main"))
	function := store.ImageStoreFunctionAppend(imageGid, name, 8)
	instruction := make([]byte, ImageInstructionSize)
	instruction[0] = byte(OpcodeReturn)
	if function != 0 || store.ImageStoreFunctionInstructionAppend(imageGid, function, instruction) != 0 || store.ImageStoreEntryMainFunctionSet(imageGid, function) != 0 {
		t.Fatalf("the image does not build")
	}
	if problem := ImageCheck(store.imageRead(nil, imageGid), make([]uint64, 1)); problem != ImageCheckProblemNone {
		t.Fatalf("the built image is refused: %s", ImageCheckProblemMessages[problem])
	}
	running := store.ImageStoreImageCurrentGet(&current)
	writes := func(target int64) []int32 {
		return []int32{
			store.ImageStoreSegmentWrite(target, 0, instruction[:4]),
			store.ImageStoreFunctionAppend(target, name, 0),
			store.ImageStoreFunctionInstructionWrite(target, 0, 0, instruction),
			store.ImageStoreFunctionInstructionAppend(target, 0, instruction),
			store.ImageStoreEntryMainFunctionSet(target, 0),
		}
	}
	for i, result := range writes(running) {
		if result != -1 {
			t.Errorf("write %d to the running image gives %d", i, result)
		}
	}
	if store.ImageStoreRunBegin(&current, imageGid) == nil {
		t.Fatal("the image does not run")
	}
	for i, result := range writes(imageGid) {
		if result != -1 {
			t.Errorf("write %d to an image that runs gives %d", i, result)
		}
	}
	if store.ImageStoreRunBegin(&current, imageGid) != nil {
		t.Errorf("an image that runs starts again")
	}
	store.ImageStoreRunEnd(imageGid)
	if store.ImageStoreFunctionInstructionAppend(imageGid, 0, instruction) != 1 {
		t.Errorf("the image is not writable after its run")
	}
}

// TestImageStoreInstructionAppendMoves appends to a function that is not
// the last to grow, which moves its code to the end.
func TestImageStoreInstructionAppendMoves(t *testing.T) {
	store := ImageStoreNew(irStoreTestRoom)
	imageGid := store.ImageStoreImageCreate()
	name := store.ImageStoreStringPoolTextAppend([]byte("f"))
	first, second := store.ImageStoreFunctionAppend(imageGid, name, 0), store.ImageStoreFunctionAppend(imageGid, name, 0)
	instruction := make([]byte, ImageInstructionSize)
	for _, step := range []struct {
		function int32
		opcode   Opcode
		pc       int32
	}{{first, OpcodeJump, 0}, {second, OpcodeReturn, 0}, {first, OpcodeReturn, 1}, {second, OpcodeZero, 1}} {
		instruction[0] = byte(step.opcode)
		if pc := store.ImageStoreFunctionInstructionAppend(imageGid, step.function, instruction); pc != step.pc {
			t.Fatalf("append to %d gives pc %d, want %d", step.function, pc, step.pc)
		}
	}
	image := store.imageRead(nil, imageGid)
	if code := image.ImageFuncs[first].FuncCode; len(code) != 2 || code[0].InstrOpcode != OpcodeJump || code[1].InstrOpcode != OpcodeReturn {
		t.Errorf("first function holds %+v", code)
	}
	if code := image.ImageFuncs[second].FuncCode; len(code) != 2 || code[0].InstrOpcode != OpcodeReturn || code[1].InstrOpcode != OpcodeZero {
		t.Errorf("second function holds %+v", code)
	}
	for store.ImageStoreFunctionInstructionAppend(imageGid, first, instruction) >= 0 {
	}
	if store.slotWritable(imageGid).slotInstructionCount > irStoreTestRoom.ImageStoreRoomInstructions {
		t.Errorf("the instructions pass their room")
	}
}

// TestImageCheckRefusals breaks a valid image in each way ImageCheck
// refuses.
func TestImageCheckRefusals(t *testing.T) {
	valid := func() Image {
		return Image{
			ImageFuncs: []Func{
				{FuncCode: []Instr{
					{InstrOpcode: OpcodeCallFrame, InstrN: 8},
					{InstrOpcode: OpcodeCallArg, InstrA: 0, InstrC: 0, InstrN: 4, InstrMode: InstrModeFrameA},
					{InstrOpcode: OpcodeCall, InstrM: 1},
					{InstrOpcode: OpcodeCallResult, InstrA: 4, InstrC: 4, InstrN: 4, InstrMode: InstrModeFrameC},
					{InstrOpcode: OpcodeJumpIf, InstrA: 0, InstrM: 5, InstrN: 5, InstrMode: InstrModeFrameA},
					{InstrOpcode: OpcodeReturn},
				}, FuncFrameSize: 8},
				{FuncCode: []Instr{{InstrOpcode: OpcodeReturn}}, FuncFrameSize: 8},
			},
			ImageStackSize: 64, ImageDataStart: 64, ImageData: make([]byte, 16),
			ImageEntryMain: 0, ImageEntryInit: ImageFuncNone,
		}
	}
	labels := make([]uint64, 1)
	image := valid()
	if problem := ImageCheck(&image, labels); problem != ImageCheckProblemNone {
		t.Fatalf("the valid image is refused: %s", ImageCheckProblemMessages[problem])
	}
	if allocs := testing.AllocsPerRun(10, func() { ImageCheck(&image, labels) }); allocs != 0 {
		t.Errorf("ImageCheck allocates %.0f times", allocs)
	}
	for _, broken := range []struct {
		name   string
		mutate func(image *Image)
		want   ImageCheckProblem
	}{
		{"data inside the stack", func(image *Image) { image.ImageDataStart = 32 }, ImageCheckProblemLayout},
		{"no main", func(image *Image) { image.ImageEntryMain = 7 }, ImageCheckProblemEntry},
		{"main without code", func(image *Image) { image.ImageFuncs[0].FuncCode = nil }, ImageCheckProblemEntry},
		{"lines outside the table", func(image *Image) { image.ImageFuncs[0].FuncLineCount = 1 }, ImageCheckProblemTables},
		{"a frame larger than the stack", func(image *Image) { image.ImageFuncs[0].FuncFrameSize = 65 }, ImageCheckProblemTables},
		{"opcode 0", func(image *Image) { image.ImageFuncs[0].FuncCode[4].InstrOpcode = OpcodeInvalid }, ImageCheckProblemOpcode},
		{"an opcode past the last", func(image *Image) { image.ImageFuncs[0].FuncCode[4].InstrOpcode = OpcodeCount }, ImageCheckProblemOpcode},
		{"no return at the end", func(image *Image) { image.ImageFuncs[0].FuncCode[5].InstrOpcode = OpcodeZero }, ImageCheckProblemEnd},
		{"a jump past the end", func(image *Image) { image.ImageFuncs[0].FuncCode[4].InstrN = 6 }, ImageCheckProblemJump},
		{"a call without its frame", func(image *Image) { image.ImageFuncs[0].FuncCode[0].InstrOpcode = OpcodeZero }, ImageCheckProblemCall},
		{"a frame smaller than the callee", func(image *Image) { image.ImageFuncs[0].FuncCode[0].InstrN = 4 }, ImageCheckProblemCall},
		{"a jump into the call", func(image *Image) { image.ImageFuncs[0].FuncCode[4].InstrM = 2 }, ImageCheckProblemCall},
		{"a call of a function without code", func(image *Image) { image.ImageFuncs[1].FuncCode = nil }, ImageCheckProblemCall},
		{"a result outside the frame of the call", func(image *Image) { image.ImageFuncs[0].FuncCode[3].InstrA = 8 }, ImageCheckProblemCall},
		{"an operand outside the frame", func(image *Image) { image.ImageFuncs[0].FuncCode[3].InstrC = 6 }, ImageCheckProblemOperand},
		{"an operand outside the memory", func(image *Image) {
			image.ImageFuncs[0].FuncCode[4].InstrMode, image.ImageFuncs[0].FuncCode[4].InstrA = 0, 80
		}, ImageCheckProblemOperand},
		{"text it does not hold", func(image *Image) {
			image.ImageFuncs[0].FuncCode[4] = Instr{InstrOpcode: OpcodePrintText, InstrN: 0, InstrM: 1}
		}, ImageCheckProblemText},
		{"a library opcode it does not enable", func(image *Image) {
			image.ImageFuncs[0].FuncCode[2].InstrOpcode = OpcodeTerminalArgCount
			image.ImageFuncs[0].FuncCode[3].InstrA = 0
		}, ImageCheckProblemOperation},
	} {
		image := valid()
		broken.mutate(&image)
		if problem := ImageCheck(&image, labels); problem != broken.want {
			t.Errorf("%s: problem %d, want %d", broken.name, problem, broken.want)
		}
	}
	long := valid()
	long.ImageFuncs[0].FuncCode = append(make([]Instr, 64), long.ImageFuncs[0].FuncCode...)
	if problem := ImageCheck(&long, labels); problem != ImageCheckProblemSize {
		t.Errorf("a function longer than the labels: problem %d", problem)
	}
}
