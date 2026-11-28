package ir

import (
	"bytes"
	"encoding/binary"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// The tags of the handles of sys/cxir, see gid-variants D154. ImageTag is a
// library resource in the reflect, meta and cxir range 0x40 to 0x5F and
// ImageNameTag an interned pool in 0x01 to 0x1F.
const (
	ImageTag     objmodel.TypeTag = 0x43
	ImageNameTag objmodel.TypeTag = 0x12
)

// imageStoreGeneration is the generation of every Gid of an ImageStore,
// which never releases an image or a name.
const imageStoreGeneration = 1

// The sizes of the structs sys/cxir declares, packed as CX packs a struct.
const (
	ImageInstructionSize = InstrSize
	ImageSlotSize        = 8
	ImageLocalSize       = 20
	ImageLineSize        = 12
)

// ImageStoreStackSizeCreated is the stack of an image CxProgramImageCreate
// makes. Its data segment starts after it.
const ImageStoreStackSizeCreated = 64 << 10

// ImageStoreRoom bounds an ImageStore, fixed before the run so no call
// allocates (decision D153). Every count but the names is per image.
type ImageStoreRoom struct {
	ImageStoreRoomImages       int
	ImageStoreRoomFunctions    int
	ImageStoreRoomInstructions int
	ImageStoreRoomSlots        int
	ImageStoreRoomLines        int
	ImageStoreRoomLocals       int
	ImageStoreRoomSpans        int
	ImageStoreRoomFileNames    int
	ImageStoreRoomLocalNames   int
	ImageStoreRoomSegmentBytes int
	ImageStoreRoomTextBytes    int
	ImageStoreRoomNames        int
	ImageStoreRoomNameBytes    int
}

// ImageStoreRoomDefault is the room of a program that calls sys/cxir, about
// 2 MiB.
var ImageStoreRoomDefault = ImageStoreRoom{
	ImageStoreRoomImages:       4,
	ImageStoreRoomFunctions:    256,
	ImageStoreRoomInstructions: 8192,
	ImageStoreRoomSlots:        1024,
	ImageStoreRoomLines:        4096,
	ImageStoreRoomLocals:       4096,
	ImageStoreRoomSpans:        256,
	ImageStoreRoomFileNames:    64,
	ImageStoreRoomLocalNames:   1024,
	ImageStoreRoomSegmentBytes: 64 << 10,
	ImageStoreRoomTextBytes:    16 << 10,
	ImageStoreRoomNames:        4096,
	ImageStoreRoomNameBytes:    64 << 10,
}

// imageStoreSlot is one writable image and the fixed arrays its tables are
// slices of.
type imageStoreSlot struct {
	slotImage            Image
	slotFunctions        []Func
	slotInstructions     []Instr
	slotInstructionCount int
	slotSlots            []Slot
	slotSlotCount        int
	slotLines            []Line
	slotLocals           []Local
	slotSpans            []Span
	slotNames            imageStoreNames
	slotNameRoom         imageStoreNames
	slotSegment          []byte
	slotText             []byte
	slotRunning          bool
}

// ImageStore holds the images a CX program builds or decodes with sys/cxir
// and the names it reads, all in room fixed before the run (decision D153).
// Gid index 0 is the image that runs, which is read only. The images
// from index 1 are writable until they run.
type ImageStore struct {
	ImageStoreRoom         ImageStoreRoom
	imageStoreSlots        []imageStoreSlot
	imageStoreCount        int
	imageStoreNameEntries  []imageStoreNameEntry
	imageStoreNameBytes    []byte
	imageStoreNameUsed     int
	imageStoreOpcodeNames  [OpcodeCount]int64
	imageStoreCurrent      *Image
	imageStoreCurrentNames imageStoreNames
}

// imageStoreNames are the names of the file, function and variable name
// tables of an image, by name Gid, 0 for an empty name. The store reads
// and writes them in place of the strings of Image, so the run path has no
// string (STRHOT, D139).
type imageStoreNames struct {
	namesFile     []int64
	namesFunction []int64
	namesLocal    []int64
}

// imageStoreNameEntry is where the text of a name lies in the name bytes.
type imageStoreNameEntry struct {
	nameEntryOffset int32
	nameEntryLength int32
}

// ImageStoreNew returns a store with the room of room.
func ImageStoreNew(room ImageStoreRoom) *ImageStore {
	store := &ImageStore{ImageStoreRoom: room}
	store.imageStoreSlots = make([]imageStoreSlot, room.ImageStoreRoomImages)
	for i := range store.imageStoreSlots {
		slot := &store.imageStoreSlots[i]
		slot.slotFunctions = make([]Func, room.ImageStoreRoomFunctions)
		slot.slotInstructions = make([]Instr, room.ImageStoreRoomInstructions)
		slot.slotSlots = make([]Slot, room.ImageStoreRoomSlots)
		slot.slotLines = make([]Line, room.ImageStoreRoomLines)
		slot.slotLocals = make([]Local, room.ImageStoreRoomLocals)
		slot.slotSpans = make([]Span, room.ImageStoreRoomSpans)
		slot.slotNameRoom = imageStoreNames{
			namesFile:     make([]int64, room.ImageStoreRoomFileNames),
			namesFunction: make([]int64, room.ImageStoreRoomFunctions),
			namesLocal:    make([]int64, room.ImageStoreRoomLocalNames),
		}
		slot.slotSegment = make([]byte, room.ImageStoreRoomSegmentBytes)
		slot.slotText = make([]byte, room.ImageStoreRoomTextBytes)
	}
	store.imageStoreNameEntries = make([]imageStoreNameEntry, 0, room.ImageStoreRoomNames)
	store.imageStoreNameBytes = make([]byte, room.ImageStoreRoomNameBytes)
	for op, name := range OpcodeNameTable {
		store.imageStoreOpcodeNames[op] = store.nameIntern([]byte(name))
	}
	return store
}

// ImageStoreCurrentSet names image as the image that runs, before the run,
// and takes its names into the store.
func (store *ImageStore) ImageStoreCurrentSet(image *Image) {
	store.imageStoreCurrent = image
	intern := func(strs []string) []int64 {
		names := make([]int64, len(strs))
		for i, str := range strs {
			names[i] = store.nameIntern([]byte(str))
		}
		return names
	}
	store.imageStoreCurrentNames = imageStoreNames{namesFile: intern(image.ImageFileNames), namesFunction: intern(image.ImageFuncNames), namesLocal: intern(image.ImageLocalNames)}
}

// namesOf returns the names of image, the running image or one of the
// store, nil for another.
func (store *ImageStore) namesOf(image *Image) *imageStoreNames {
	if image != nil && image == store.imageStoreCurrent {
		return &store.imageStoreCurrentNames
	}
	for i := range store.imageStoreSlots[:store.imageStoreCount] {
		if image == &store.imageStoreSlots[i].slotImage {
			return &store.imageStoreSlots[i].slotNames
		}
	}
	return nil
}

// imageGidCompose returns the image Gid of index, 0 for the running image.
func imageGidCompose(index int) int64 {
	return int64(uint64(ImageTag)<<objmodel.GidTypeTagShift | uint64(imageStoreGeneration)<<objmodel.GidGenerationShift | uint64(uint32(index)))
}

// imageGidIndex returns the index of imageGid, false when it is no
// image Gid of store.
func (store *ImageStore) imageGidIndex(imageGid int64) (int, bool) {
	if imageGid <= 0 || objmodel.GidTypeTag(imageGid) != ImageTag || objmodel.GidGeneration(imageGid) != imageStoreGeneration {
		return 0, false
	}
	index := int64(objmodel.GidIndex(imageGid))
	return int(index), index <= int64(store.imageStoreCount)
}

// imageRead returns the image of imageGid, current for index 0, nil when
// there is none.
func (store *ImageStore) imageRead(current *Image, imageGid int64) *Image {
	index, found := store.imageGidIndex(imageGid)
	switch {
	case !found:
		return nil
	case index == 0:
		return current
	}
	return &store.imageStoreSlots[index-1].slotImage
}

// slotWritable returns the writable image of imageGid, nil for the running
// image, an image that runs or no image.
func (store *ImageStore) slotWritable(imageGid int64) *imageStoreSlot {
	index, found := store.imageGidIndex(imageGid)
	if !found || index == 0 || store.imageStoreSlots[index-1].slotRunning {
		return nil
	}
	return &store.imageStoreSlots[index-1]
}

// ImageStoreRunBegin marks the image of imageGid as running, so no call
// changes it, and returns it, nil for no image or one that already runs.
// The running image of the caller is current.
func (store *ImageStore) ImageStoreRunBegin(current *Image, imageGid int64) *Image {
	index, found := store.imageGidIndex(imageGid)
	if !found || index == 0 || store.imageStoreSlots[index-1].slotRunning {
		return nil
	}
	slot := &store.imageStoreSlots[index-1]
	if &slot.slotImage == current {
		return nil
	}
	slot.slotRunning = true
	return &slot.slotImage
}

// ImageStoreRunEnd makes the image of imageGid writable again.
func (store *ImageStore) ImageStoreRunEnd(imageGid int64) {
	if index, found := store.imageGidIndex(imageGid); found && index > 0 {
		store.imageStoreSlots[index-1].slotRunning = false
	}
}

// nameIntern returns the name Gid of text, the name already in the pool or
// a new one, 0 when text is empty or the pool is full.
func (store *ImageStore) nameIntern(text []byte) int64 {
	if len(text) == 0 {
		return 0
	}
	for index, entry := range store.imageStoreNameEntries {
		if bytes.Equal(store.imageStoreNameBytes[entry.nameEntryOffset:entry.nameEntryOffset+entry.nameEntryLength], text) {
			return nameGidCompose(index)
		}
	}
	used := store.imageStoreNameUsed
	if len(store.imageStoreNameEntries) == cap(store.imageStoreNameEntries) || used+len(text) > len(store.imageStoreNameBytes) {
		return 0
	}
	copy(store.imageStoreNameBytes[used:], text)
	store.imageStoreNameUsed += len(text)
	store.imageStoreNameEntries = append(store.imageStoreNameEntries, imageStoreNameEntry{nameEntryOffset: int32(used), nameEntryLength: int32(len(text))})
	return nameGidCompose(len(store.imageStoreNameEntries) - 1)
}

// nameGidCompose returns the name Gid of pool entry index.
func nameGidCompose(index int) int64 {
	return int64(uint64(ImageNameTag)<<objmodel.GidTypeTagShift | uint64(imageStoreGeneration)<<objmodel.GidGenerationShift | uint64(uint32(index)))
}

// nameBytes returns the text of a name Gid, empty for 0, false when it is
// no name of store.
func (store *ImageStore) nameBytes(name int64) ([]byte, bool) {
	if name == 0 {
		return nil, true
	}
	if name < 0 || objmodel.GidTypeTag(name) != ImageNameTag || objmodel.GidGeneration(name) != imageStoreGeneration {
		return nil, false
	}
	index := int64(objmodel.GidIndex(name))
	if index >= int64(len(store.imageStoreNameEntries)) {
		return nil, false
	}
	entry := store.imageStoreNameEntries[index]
	return store.imageStoreNameBytes[entry.nameEntryOffset : entry.nameEntryOffset+entry.nameEntryLength], true
}

// ImageStoreStringPoolTextAppend returns the name of text, equal for equal
// texts, 0 when text is empty or the pool is full.
func (store *ImageStore) ImageStoreStringPoolTextAppend(text []byte) int64 {
	return store.nameIntern(text)
}

// ImageStoreStringPoolTextRead copies the text of name into buffer and
// returns the bytes copied, -1 for no name. A longer name is cut at the
// size of buffer.
func (store *ImageStore) ImageStoreStringPoolTextRead(name int64, buffer []byte) int32 {
	text, found := store.nameBytes(name)
	if !found || name == 0 {
		return -1
	}
	return int32(copy(buffer, text))
}

// ImageStoreImageCurrentGet returns the Gid of the running image, 0 when
// the program runs without one.
func (store *ImageStore) ImageStoreImageCurrentGet(current *Image) int64 {
	if current == nil {
		return 0
	}
	return imageGidCompose(0)
}

// ImageStoreImageCreate returns the Gid of a new empty writable image
// with no entry, 0 when the store is full.
func (store *ImageStore) ImageStoreImageCreate() int64 {
	slot := store.slotNext()
	if slot == nil {
		return 0
	}
	image := &slot.slotImage
	image.ImageStackSize = ImageStoreStackSizeCreated
	image.ImageDataStart = ImageStoreStackSizeCreated
	return store.slotCommit()
}

// slotNext returns the next free slot, reset to an empty image, nil when
// the store is full.
func (store *ImageStore) slotNext() *imageStoreSlot {
	if store.imageStoreCount >= len(store.imageStoreSlots) {
		return nil
	}
	slot := &store.imageStoreSlots[store.imageStoreCount]
	slot.slotImage = Image{
		ImageFuncs:     slot.slotFunctions[:0],
		ImageData:      slot.slotSegment[:0],
		ImageText:      slot.slotText[:0],
		ImageEntryInit: ImageFuncNone,
		ImageEntryMain: ImageFuncNone,
		ImageLines:     slot.slotLines[:0],
		ImageLocals:    slot.slotLocals[:0],
		ImageReadOnly:  slot.slotSpans[:0],
	}
	slot.slotNames = imageStoreNames{namesFile: slot.slotNameRoom.namesFile[:0], namesFunction: slot.slotNameRoom.namesFunction[:0], namesLocal: slot.slotNameRoom.namesLocal[:0]}
	slot.slotInstructionCount, slot.slotSlotCount = 0, 0
	return slot
}

// slotCommit counts the slot slotNext returned and returns its Gid.
func (store *ImageStore) slotCommit() int64 {
	store.imageStoreCount++
	return imageGidCompose(store.imageStoreCount)
}

// ImageStoreFunctionCountGet returns the functions of image, -1 for no
// image.
func (store *ImageStore) ImageStoreFunctionCountGet(current *Image, imageGid int64) int32 {
	if image := store.imageRead(current, imageGid); image != nil {
		return int32(len(image.ImageFuncs))
	}
	return -1
}

// ImageStoreEntryMainFunctionGet returns the main entry of image, -1 for
// none or no image.
func (store *ImageStore) ImageStoreEntryMainFunctionGet(current *Image, imageGid int64) int32 {
	if image := store.imageRead(current, imageGid); image != nil {
		return int32(image.ImageEntryMain)
	}
	return -1
}

// ImageStoreEntryInitFunctionGet returns the *init entry of image, -1 for
// none or no image.
func (store *ImageStore) ImageStoreEntryInitFunctionGet(current *Image, imageGid int64) int32 {
	if image := store.imageRead(current, imageGid); image != nil {
		return int32(image.ImageEntryInit)
	}
	return -1
}

// ImageStoreEntryMainFunctionSet makes function the main entry of a
// writable image and returns 0, -1 when it is no function of it.
func (store *ImageStore) ImageStoreEntryMainFunctionSet(imageGid int64, function int32) int32 {
	slot := store.slotWritable(imageGid)
	if slot == nil || function < 0 || int(function) >= len(slot.slotImage.ImageFuncs) {
		return -1
	}
	slot.slotImage.ImageEntryMain = uint32(function)
	return 0
}

// ImageStoreStackSizeInBytesGet returns the stack of image, -1 for no image.
func (store *ImageStore) ImageStoreStackSizeInBytesGet(current *Image, imageGid int64) int32 {
	if image := store.imageRead(current, imageGid); image != nil {
		return int32(image.ImageStackSize)
	}
	return -1
}

// ImageStoreSegmentStartGet returns the address of the data segment of image,
// -1 for no image.
func (store *ImageStore) ImageStoreSegmentStartGet(current *Image, imageGid int64) int32 {
	if image := store.imageRead(current, imageGid); image != nil {
		return int32(image.ImageDataStart)
	}
	return -1
}

// ImageStoreSegmentRead copies the data segment of image from offset into
// buffer and returns the bytes copied, -1 for no image or an offset past
// its end. It reads the segment as the image loads it.
func (store *ImageStore) ImageStoreSegmentRead(current *Image, imageGid int64, offset int32, buffer []byte) int32 {
	image := store.imageRead(current, imageGid)
	if image == nil || offset < 0 || int(offset) > len(image.ImageData) {
		return -1
	}
	return int32(copy(buffer, image.ImageData[offset:]))
}

// ImageStoreSegmentWrite copies buffer into the data segment of a writable
// image at offset, which grows the segment up to its room, and returns the
// bytes written, -1 when it does not fit.
func (store *ImageStore) ImageStoreSegmentWrite(imageGid int64, offset int32, buffer []byte) int32 {
	slot := store.slotWritable(imageGid)
	if slot == nil || offset < 0 || int64(offset)+int64(len(buffer)) > int64(len(slot.slotSegment)) {
		return -1
	}
	image := &slot.slotImage
	if end := int(offset) + len(buffer); end > len(image.ImageData) {
		clear(slot.slotSegment[len(image.ImageData):end])
		image.ImageData = slot.slotSegment[:end]
	}
	return int32(copy(image.ImageData[offset:], buffer))
}

// ImageStoreFileEncode writes image into buffer as ImageFileEncode does and
// returns the size, -1 for no image or a buffer too short.
func (store *ImageStore) ImageStoreFileEncode(current *Image, imageGid int64, buffer []byte) int32 {
	image := store.imageRead(current, imageGid)
	if image == nil {
		return -1
	}
	names := store.namesOf(image)
	if names == nil {
		return -1
	}
	writer := imageFileWriter{}
	store.imageEncode(&writer, image, names)
	size := writer.imageFileWriterAt
	if len(buffer) < size {
		return -1
	}
	writer = imageFileWriter{imageFileWriterBytes: buffer}
	store.imageEncode(&writer, image, names)
	return int32(size)
}

// imageEncode writes image as ImageFileEncode does, its name tables from
// names.
func (store *ImageStore) imageEncode(writer *imageFileWriter, image *Image, names *imageStoreNames) {
	writer.imageWriteHead(image)
	store.namesWrite(writer, names.namesFile)
	store.namesWrite(writer, names.namesFunction)
	writer.imageWriteLocals(image)
	store.namesWrite(writer, names.namesLocal)
	writer.imageWriteTail(image)
}

// namesWrite writes a name table as stringsWrite writes the strings.
func (store *ImageStore) namesWrite(writer *imageFileWriter, names []int64) {
	writer.uint32Write(uint32(len(names)))
	for _, name := range names {
		text, _ := store.nameBytes(name)
		writer.uint32Write(uint32(len(text)))
		if writer.imageFileWriterBytes != nil {
			copy(writer.imageFileWriterBytes[writer.imageFileWriterAt:], text)
		}
		writer.imageFileWriterAt += len(text)
	}
}

// ImageStoreFileDecode decodes the image file in file into a new writable
// image and returns its Gid, -1 for a file ImageFileDecode refuses or
// one that does not fit the room.
func (store *ImageStore) ImageStoreFileDecode(file []byte) int64 {
	slot := store.slotNext()
	if slot == nil || !store.slotDecode(slot, file) {
		return -1
	}
	return store.slotCommit()
}

// slotDecode reads the fields ImageFileEncode writes into the arrays of
// slot, false when the file is refused or does not fit.
func (store *ImageStore) slotDecode(slot *imageStoreSlot, file []byte) bool {
	if len(file) < len(imageFileMagicBytes) || !bytes.Equal(file[:len(imageFileMagicBytes)], imageFileMagicBytes[:]) {
		return false
	}
	image := &slot.slotImage
	reader := &imageFileReader{imageFileReaderBytes: file[len(imageFileMagicBytes):]}
	if reader.uint32Read() != ImageFileVersion || reader.uint32Read() != uint32(OpcodeCount) || reader.uint32Read() != InstrSize {
		return false
	}
	functions := int(reader.countRead(4))
	if functions > len(slot.slotFunctions) {
		return false
	}
	image.ImageFuncs = slot.slotFunctions[:functions]
	for i := range image.ImageFuncs {
		fn := &image.ImageFuncs[i]
		count := int(reader.countRead(InstrSize))
		if slot.slotInstructionCount+count > len(slot.slotInstructions) {
			return false
		}
		fn.FuncCode = slot.slotInstructions[slot.slotInstructionCount : slot.slotInstructionCount+count : slot.slotInstructionCount+count]
		slot.slotInstructionCount += count
		for j := range fn.FuncCode {
			in := &fn.FuncCode[j]
			in.InstrOpcode, in.InstrMode = Opcode(reader.uint16Read()), InstrMode(reader.uint16Read())
			in.InstrA, in.InstrB, in.InstrC, in.InstrD = reader.uint32Read(), reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
			in.InstrN, in.InstrM, in.InstrAux = reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
		}
		fn.FuncFrameSize = reader.uint32Read()
		var fits bool
		if fn.FuncInputs, fits = slot.slotsRead(reader); !fits {
			return false
		}
		if fn.FuncOutputs, fits = slot.slotsRead(reader); !fits {
			return false
		}
		fn.FuncLineFirst, fn.FuncLineCount = reader.uint32Read(), reader.uint32Read()
		fn.FuncLocalFirst, fn.FuncLocalCount = reader.uint32Read(), reader.uint32Read()
	}
	segment := reader.bytesRead()
	image.ImageDataStart, image.ImageStackSize, image.ImageScratchSize = reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
	text := reader.bytesRead()
	if len(segment) > len(slot.slotSegment) || len(text) > len(slot.slotText) {
		return false
	}
	image.ImageData = slot.slotSegment[:copy(slot.slotSegment, segment)]
	image.ImageText = slot.slotText[:copy(slot.slotText, text)]
	image.ImageEntryInit, image.ImageEntryMain = reader.uint32Read(), reader.uint32Read()
	lines := int(reader.countRead(12))
	if lines > len(slot.slotLines) {
		return false
	}
	image.ImageLines = slot.slotLines[:lines]
	for i := range image.ImageLines {
		image.ImageLines[i] = Line{LinePc: reader.uint32Read(), LineFile: reader.uint32Read(), LineNumber: reader.uint32Read()}
	}
	var fits bool
	if slot.slotNames.namesFile, fits = store.namesRead(reader, slot.slotNameRoom.namesFile); !fits {
		return false
	}
	if slot.slotNames.namesFunction, fits = store.namesRead(reader, slot.slotNameRoom.namesFunction); !fits {
		return false
	}
	locals := int(reader.countRead(20))
	if locals > len(slot.slotLocals) {
		return false
	}
	image.ImageLocals = slot.slotLocals[:locals]
	for i := range image.ImageLocals {
		local := &image.ImageLocals[i]
		local.LocalOffset, local.LocalSize = reader.uint32Read(), reader.uint32Read()
		local.LocalName, local.LocalTypeName = reader.uint32Read(), reader.uint32Read()
		local.LocalType = valuetypes.Code(int32(reader.uint32Read()))
	}
	if slot.slotNames.namesLocal, fits = store.namesRead(reader, slot.slotNameRoom.namesLocal); !fits {
		return false
	}
	spans := int(reader.countRead(8))
	if spans > len(slot.slotSpans) {
		return false
	}
	image.ImageReadOnly = slot.slotSpans[:spans]
	for i := range image.ImageReadOnly {
		image.ImageReadOnly[i] = Span{SpanStart: reader.uint32Read(), SpanEnd: reader.uint32Read()}
	}
	if reader.uint32Read() != opcodetable.OpcodeSetWordCount {
		return false
	}
	for i := range image.ImageOperationsEnabled {
		image.ImageOperationsEnabled[i] = reader.uint64Read()
	}
	mode := reader.uint32Read()
	if mode >= uint32(output.ModeCount) {
		return false
	}
	image.ImageOutputMode = output.Mode(mode)
	return !reader.imageFileReaderShort && len(reader.imageFileReaderBytes) == 0
}

// slotsRead reads a slot table into the slot array of slot.
func (slot *imageStoreSlot) slotsRead(reader *imageFileReader) ([]Slot, bool) {
	count := int(reader.countRead(8))
	if slot.slotSlotCount+count > len(slot.slotSlots) {
		return nil, false
	}
	slots := slot.slotSlots[slot.slotSlotCount : slot.slotSlotCount+count : slot.slotSlotCount+count]
	slot.slotSlotCount += count
	for i := range slots {
		slots[i] = Slot{SlotOffset: reader.uint32Read(), SlotSize: reader.uint32Read()}
	}
	return slots, true
}

// namesRead reads a string table into room as names of store.
func (store *ImageStore) namesRead(reader *imageFileReader, room []int64) ([]int64, bool) {
	count := int(reader.countRead(4))
	if count > len(room) {
		return nil, false
	}
	names := room[:count]
	for i := range names {
		text := reader.bytesRead()
		names[i] = store.nameIntern(text)
		if names[i] == 0 && len(text) > 0 {
			return nil, false
		}
	}
	return names, true
}

// functionRead returns function of image, nil when it has none.
func functionRead(image *Image, function int32) *Func {
	if image == nil || function < 0 || int(function) >= len(image.ImageFuncs) {
		return nil
	}
	return &image.ImageFuncs[function]
}

// ImageStoreFunctionNameGet returns the name of function, 0 for none.
func (store *ImageStore) ImageStoreFunctionNameGet(current *Image, imageGid int64, function int32) int64 {
	image := store.imageRead(current, imageGid)
	names := store.namesOf(image)
	if functionRead(image, function) == nil || names == nil || int(function) >= len(names.namesFunction) {
		return 0
	}
	return names.namesFunction[function]
}

// ImageStoreFunctionFrameSizeInBytesGet returns the frame of function, -1
// for none.
func (store *ImageStore) ImageStoreFunctionFrameSizeInBytesGet(current *Image, imageGid int64, function int32) int32 {
	if fn := functionRead(store.imageRead(current, imageGid), function); fn != nil {
		return int32(fn.FuncFrameSize)
	}
	return -1
}

// ImageStoreFunctionInputCountGet returns the inputs of function, -1 for
// none.
func (store *ImageStore) ImageStoreFunctionInputCountGet(current *Image, imageGid int64, function int32) int32 {
	if fn := functionRead(store.imageRead(current, imageGid), function); fn != nil {
		return int32(len(fn.FuncInputs))
	}
	return -1
}

// ImageStoreFunctionOutputCountGet returns the outputs of function, -1 for
// none.
func (store *ImageStore) ImageStoreFunctionOutputCountGet(current *Image, imageGid int64, function int32) int32 {
	if fn := functionRead(store.imageRead(current, imageGid), function); fn != nil {
		return int32(len(fn.FuncOutputs))
	}
	return -1
}

// ImageStoreFunctionLocalCountGet returns the variables of function, -1
// for none.
func (store *ImageStore) ImageStoreFunctionLocalCountGet(current *Image, imageGid int64, function int32) int32 {
	if fn := functionRead(store.imageRead(current, imageGid), function); fn != nil {
		return int32(fn.FuncLocalCount)
	}
	return -1
}

// slotWrite writes slot as a CxFrameSlot into buffer and returns 0, -1 when
// buffer is shorter.
func slotWrite(buffer []byte, slot Slot) int32 {
	if len(buffer) < ImageSlotSize {
		return -1
	}
	binary.LittleEndian.PutUint32(buffer, slot.SlotOffset)
	binary.LittleEndian.PutUint32(buffer[4:], slot.SlotSize)
	return 0
}

// ImageStoreFunctionInputSlotRead writes input ordinal of function into the
// CxFrameSlot in buffer and returns 0, -1 for none.
func (store *ImageStore) ImageStoreFunctionInputSlotRead(current *Image, imageGid int64, function int32, ordinal int32, buffer []byte) int32 {
	fn := functionRead(store.imageRead(current, imageGid), function)
	if fn == nil || ordinal < 0 || int(ordinal) >= len(fn.FuncInputs) {
		return -1
	}
	return slotWrite(buffer, fn.FuncInputs[ordinal])
}

// ImageStoreFunctionOutputSlotRead writes output ordinal of function into
// the CxFrameSlot in buffer and returns 0, -1 for none.
func (store *ImageStore) ImageStoreFunctionOutputSlotRead(current *Image, imageGid int64, function int32, ordinal int32, buffer []byte) int32 {
	fn := functionRead(store.imageRead(current, imageGid), function)
	if fn == nil || ordinal < 0 || int(ordinal) >= len(fn.FuncOutputs) {
		return -1
	}
	return slotWrite(buffer, fn.FuncOutputs[ordinal])
}

// ImageStoreFunctionLocalRead writes variable ordinal of function into the
// CxFrameLocal in buffer and returns 0, -1 for none.
func (store *ImageStore) ImageStoreFunctionLocalRead(current *Image, imageGid int64, function int32, ordinal int32, buffer []byte) int32 {
	image := store.imageRead(current, imageGid)
	fn := functionRead(image, function)
	if fn == nil || ordinal < 0 || uint32(ordinal) >= fn.FuncLocalCount || len(buffer) < ImageLocalSize {
		return -1
	}
	index := uint64(fn.FuncLocalFirst) + uint64(ordinal)
	if index >= uint64(len(image.ImageLocals)) {
		return -1
	}
	local := &image.ImageLocals[index]
	binary.LittleEndian.PutUint32(buffer, local.LocalOffset)
	binary.LittleEndian.PutUint32(buffer[4:], local.LocalSize)
	binary.LittleEndian.PutUint32(buffer[8:], local.LocalName)
	binary.LittleEndian.PutUint32(buffer[12:], local.LocalTypeName)
	binary.LittleEndian.PutUint32(buffer[16:], uint32(int32(local.LocalType)))
	return 0
}

// ImageStoreLocalNameGet returns name localName of the variable name table
// of image, 0 for none.
func (store *ImageStore) ImageStoreLocalNameGet(current *Image, imageGid int64, localName int32) int64 {
	image := store.imageRead(current, imageGid)
	names := store.namesOf(image)
	if names == nil || localName < 0 || int(localName) >= len(names.namesLocal) {
		return 0
	}
	return names.namesLocal[localName]
}

// ImageStoreFunctionAppend adds a function named name with a frame of
// frameSize bytes and no code to a writable image and returns its id, -1
// when the image is full or name is no name.
func (store *ImageStore) ImageStoreFunctionAppend(imageGid int64, name int64, frameSize int32) int32 {
	slot := store.slotWritable(imageGid)
	_, named := store.nameBytes(name)
	if slot == nil || !named || frameSize < 0 {
		return -1
	}
	image := &slot.slotImage
	count := len(image.ImageFuncs)
	if count >= len(slot.slotFunctions) || len(slot.slotNames.namesFunction) != count {
		return -1
	}
	at := slot.slotInstructionCount
	slot.slotFunctions[count] = Func{FuncCode: slot.slotInstructions[at:at:at], FuncFrameSize: uint32(frameSize)}
	image.ImageFuncs = slot.slotFunctions[:count+1]
	slot.slotNameRoom.namesFunction[count] = name
	slot.slotNames.namesFunction = slot.slotNameRoom.namesFunction[:count+1]
	return int32(count)
}

// ImageStoreFunctionInstructionCountGet returns the instructions of
// function, -1 for none.
func (store *ImageStore) ImageStoreFunctionInstructionCountGet(current *Image, imageGid int64, function int32) int32 {
	if fn := functionRead(store.imageRead(current, imageGid), function); fn != nil {
		return int32(len(fn.FuncCode))
	}
	return -1
}

// ImageStoreFunctionInstructionRead writes instruction pc of function into
// the CxInstruction in buffer and returns 0, -1 for none.
func (store *ImageStore) ImageStoreFunctionInstructionRead(current *Image, imageGid int64, function int32, pc int32, buffer []byte) int32 {
	fn := functionRead(store.imageRead(current, imageGid), function)
	if fn == nil || pc < 0 || int(pc) >= len(fn.FuncCode) || len(buffer) < ImageInstructionSize {
		return -1
	}
	in := &fn.FuncCode[pc]
	binary.LittleEndian.PutUint16(buffer, uint16(in.InstrOpcode))
	binary.LittleEndian.PutUint16(buffer[2:], uint16(in.InstrMode))
	for i, operand := range [...]uint32{in.InstrA, in.InstrB, in.InstrC, in.InstrD, in.InstrN, in.InstrM, in.InstrAux} {
		binary.LittleEndian.PutUint32(buffer[4+4*i:], operand)
	}
	return 0
}

// instructionOf reads the CxInstruction in buffer, false when buffer is
// short or the opcode is invalid.
func instructionOf(buffer []byte) (Instr, bool) {
	if len(buffer) < ImageInstructionSize {
		return Instr{}, false
	}
	operand := func(i int) uint32 { return binary.LittleEndian.Uint32(buffer[4+4*i:]) }
	in := Instr{
		InstrOpcode: Opcode(binary.LittleEndian.Uint16(buffer)), InstrMode: InstrMode(binary.LittleEndian.Uint16(buffer[2:])),
		InstrA: operand(0), InstrB: operand(1), InstrC: operand(2), InstrD: operand(3), InstrN: operand(4), InstrM: operand(5), InstrAux: operand(6),
	}
	return in, in.InstrOpcode != OpcodeInvalid && in.InstrOpcode < OpcodeCount
}

// instructionSet stores in at pc of function, and enables the library
// operation of a library opcode in the image, which Run then needs the
// running program to enable (decision D119).
func instructionSet(image *Image, fn *Func, pc int, in Instr) {
	fn.FuncCode[pc] = in
	if OpcodeLibrary(in.InstrOpcode) {
		image.ImageOperationsEnabled.OpcodeSetAdd(OpcodeLibraryOperation(in.InstrOpcode))
	}
}

// ImageStoreFunctionInstructionWrite replaces instruction pc of function of
// a writable image with the CxInstruction in buffer and returns 0, -1 for
// none or an invalid opcode.
func (store *ImageStore) ImageStoreFunctionInstructionWrite(imageGid int64, function int32, pc int32, buffer []byte) int32 {
	slot := store.slotWritable(imageGid)
	if slot == nil {
		return -1
	}
	fn := functionRead(&slot.slotImage, function)
	in, valid := instructionOf(buffer)
	if fn == nil || !valid || pc < 0 || int(pc) >= len(fn.FuncCode) {
		return -1
	}
	instructionSet(&slot.slotImage, fn, int(pc), in)
	return 0
}

// ImageStoreFunctionInstructionAppend appends the CxInstruction in buffer
// to function of a writable image and returns its pc, -1 for none, an
// invalid opcode or a full image. Code of a function that is not the last
// one to grow moves to the end of the instructions of the image.
func (store *ImageStore) ImageStoreFunctionInstructionAppend(imageGid int64, function int32, buffer []byte) int32 {
	slot := store.slotWritable(imageGid)
	if slot == nil {
		return -1
	}
	fn := functionRead(&slot.slotImage, function)
	in, valid := instructionOf(buffer)
	if fn == nil || !valid {
		return -1
	}
	count := len(fn.FuncCode)
	start := slot.slotInstructionCount - count
	if count == 0 || start < 0 || &slot.slotInstructions[start] != &fn.FuncCode[0] {
		start = slot.slotInstructionCount
		if start+count+1 > len(slot.slotInstructions) {
			return -1
		}
		copy(slot.slotInstructions[start:], fn.FuncCode)
		slot.slotInstructionCount += count
	}
	if slot.slotInstructionCount >= len(slot.slotInstructions) {
		return -1
	}
	fn.FuncCode = slot.slotInstructions[start : start+count+1 : start+count+1]
	slot.slotInstructionCount++
	instructionSet(&slot.slotImage, fn, count, in)
	return int32(count)
}

// ImageStoreOpcodeNameGet returns the name of image opcode opcode, 0 for
// none.
func (store *ImageStore) ImageStoreOpcodeNameGet(opcode int32) int64 {
	if opcode < 0 || opcode >= int32(OpcodeCount) {
		return 0
	}
	return store.imageStoreOpcodeNames[opcode]
}

// ImageStoreOpcodeByNameGet returns the image opcode named name, -1 for
// none.
func (store *ImageStore) ImageStoreOpcodeByNameGet(name int64) int32 {
	if name == 0 {
		return -1
	}
	for op, opcodeName := range store.imageStoreOpcodeNames {
		if opcodeName == name {
			return int32(op)
		}
	}
	return -1
}

// ImageStoreOpcodeLibraryRequiredGet reports whether opcode is a library
// opcode, which a run must enable (decision D119).
func ImageStoreOpcodeLibraryRequiredGet(opcode int32) bool {
	return opcode >= 0 && opcode < int32(OpcodeCount) && OpcodeLibrary(Opcode(opcode))
}

// ImageStoreSourceLineAtProgramCounterRead writes the source line of pc in
// function into the CxSourceLine in buffer and returns 0, -1 for none.
func (store *ImageStore) ImageStoreSourceLineAtProgramCounterRead(current *Image, imageGid int64, function int32, pc int32, buffer []byte) int32 {
	image := store.imageRead(current, imageGid)
	fn := functionRead(image, function)
	if fn == nil || pc < 0 || len(buffer) < ImageLineSize || uint64(fn.FuncLineFirst)+uint64(fn.FuncLineCount) > uint64(len(image.ImageLines)) {
		return -1
	}
	line, found := LineAt(image, uint32(function), uint32(pc))
	if !found {
		return -1
	}
	binary.LittleEndian.PutUint32(buffer, line.LinePc)
	binary.LittleEndian.PutUint32(buffer[4:], line.LineFile)
	binary.LittleEndian.PutUint32(buffer[8:], line.LineNumber)
	return 0
}

// ImageStoreSourceFileNameGet returns the name of source file sourceFile of
// image, 0 for none.
func (store *ImageStore) ImageStoreSourceFileNameGet(current *Image, imageGid int64, sourceFile int32) int64 {
	image := store.imageRead(current, imageGid)
	names := store.namesOf(image)
	if names == nil || sourceFile < 0 || int(sourceFile) >= len(names.namesFile) {
		return 0
	}
	return names.namesFile[sourceFile]
}

// ImageOperationsWithin reports whether every library operation image
// enables is enabled in allowed, the rule for an image a program runs
// (decision D153).
func ImageOperationsWithin(image *Image, allowed *opcodetable.OpcodeSet) bool {
	for i, word := range image.ImageOperationsEnabled {
		if word&^allowed[i] != 0 {
			return false
		}
	}
	return true
}
