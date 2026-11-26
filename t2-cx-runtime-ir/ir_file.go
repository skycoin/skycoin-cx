package ir

import (
	"encoding/binary"

	"github.com/skycoin/skycoin-cx/t0-types"
	"github.com/skycoin/skycoin-cx/t1-constants"
)

// ImageFileMagic starts every image file, see docs/Microcontrollers.md.
const ImageFileMagic = "CXIMAGE\x00"

// ImageFileVersion is the layout version of an image file. Opcode numbers
// change from one cx version to the next, so a file also stores OpcodeCount
// and InstrSize, and ImageFileDecode refuses a file whose values differ.
const ImageFileVersion = 1

// ImageFileEncode returns image as bytes that ImageFileDecode reads back,
// every field little endian in declaration order.
func ImageFileEncode(image *Image) []byte {
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
	file = binary.LittleEndian.AppendUint32(file, constants.OpcodeSetWordCount)
	for _, word := range image.ImageOperationsEnabled {
		file = binary.LittleEndian.AppendUint64(file, word)
	}
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

// ImageFileProblem says why ImageFileDecode refused a file.
type ImageFileProblem uint32

const (
	// ImageFileProblemNone is a file that decoded.
	ImageFileProblemNone ImageFileProblem = iota
	// ImageFileProblemMagic is a file that does not start with ImageFileMagic.
	ImageFileProblemMagic
	// ImageFileProblemVersion is a file another cx version wrote.
	ImageFileProblemVersion
	// ImageFileProblemTruncated is a file that ends inside a field or has
	// bytes after the last one.
	ImageFileProblemTruncated
)

// ImageFileProblemMessages describes each problem. Read only.
var ImageFileProblemMessages = [...]string{
	ImageFileProblemNone:      "",
	ImageFileProblemMagic:     "not a cx image file",
	ImageFileProblemVersion:   "the image file was written by another cx version, build it again",
	ImageFileProblemTruncated: "the image file is truncated or damaged",
}

// imageFileReader reads the fields of an image file in order. A read past
// the end sets imageFileReaderShort and returns zeros from then on.
type imageFileReader struct {
	imageFileReaderBytes []byte
	imageFileReaderShort bool
}

func (reader *imageFileReader) take(size uint64) []byte {
	if reader.imageFileReaderShort || uint64(len(reader.imageFileReaderBytes)) < size {
		reader.imageFileReaderShort = true
		return nil
	}
	taken := reader.imageFileReaderBytes[:size:size]
	reader.imageFileReaderBytes = reader.imageFileReaderBytes[size:]
	return taken
}

func (reader *imageFileReader) uint16Read() uint16 {
	if bytes := reader.take(2); bytes != nil {
		return binary.LittleEndian.Uint16(bytes)
	}
	return 0
}

func (reader *imageFileReader) uint32Read() uint32 {
	if bytes := reader.take(4); bytes != nil {
		return binary.LittleEndian.Uint32(bytes)
	}
	return 0
}

func (reader *imageFileReader) uint64Read() uint64 {
	if bytes := reader.take(8); bytes != nil {
		return binary.LittleEndian.Uint64(bytes)
	}
	return 0
}

// countRead reads a count of records of recordSize bytes and returns 0,
// marking the file short, when they cannot all be in the rest of the file.
// That keeps a damaged count from allocating more than the file holds.
func (reader *imageFileReader) countRead(recordSize uint64) uint32 {
	count := reader.uint32Read()
	if uint64(count)*recordSize > uint64(len(reader.imageFileReaderBytes)) {
		reader.imageFileReaderShort = true
		return 0
	}
	return count
}

func (reader *imageFileReader) bytesRead() []byte {
	return reader.take(uint64(reader.countRead(1)))
}

func (reader *imageFileReader) slotsRead() []Slot {
	slots := make([]Slot, reader.countRead(8))
	for i := range slots {
		slots[i] = Slot{SlotOffset: reader.uint32Read(), SlotSize: reader.uint32Read()}
	}
	return slots
}

func (reader *imageFileReader) stringsRead() []string {
	strs := make([]string, reader.countRead(4))
	for i := range strs {
		strs[i] = string(reader.bytesRead())
	}
	return strs
}

// ImageFileDecode reads an image that ImageFileEncode wrote. The image keeps
// slices of file for its data and text, so file must not change after.
func ImageFileDecode(file []byte) (Image, ImageFileProblem) {
	var image Image
	if len(file) < len(ImageFileMagic) || string(file[:len(ImageFileMagic)]) != ImageFileMagic {
		return image, ImageFileProblemMagic
	}
	reader := &imageFileReader{imageFileReaderBytes: file[len(ImageFileMagic):]}
	version, opcodeCount, instrSize := reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
	if reader.imageFileReaderShort {
		return image, ImageFileProblemTruncated
	}
	if version != ImageFileVersion || opcodeCount != uint32(OpcodeCount) || instrSize != InstrSize {
		return image, ImageFileProblemVersion
	}
	image.ImageFuncs = make([]Func, reader.countRead(4))
	for i := range image.ImageFuncs {
		fn := &image.ImageFuncs[i]
		fn.FuncCode = make([]Instr, reader.countRead(InstrSize))
		for j := range fn.FuncCode {
			in := &fn.FuncCode[j]
			in.InstrOpcode, in.InstrMode = Opcode(reader.uint16Read()), InstrMode(reader.uint16Read())
			in.InstrA, in.InstrB, in.InstrC, in.InstrD = reader.uint32Read(), reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
			in.InstrN, in.InstrM, in.InstrAux = reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
		}
		fn.FuncFrameSize = reader.uint32Read()
		fn.FuncInputs, fn.FuncOutputs = reader.slotsRead(), reader.slotsRead()
		fn.FuncLineFirst, fn.FuncLineCount = reader.uint32Read(), reader.uint32Read()
		fn.FuncLocalFirst, fn.FuncLocalCount = reader.uint32Read(), reader.uint32Read()
	}
	image.ImageData = reader.bytesRead()
	image.ImageDataStart, image.ImageStackSize, image.ImageScratchSize = reader.uint32Read(), reader.uint32Read(), reader.uint32Read()
	image.ImageText = reader.bytesRead()
	image.ImageEntryInit, image.ImageEntryMain = reader.uint32Read(), reader.uint32Read()
	image.ImageLines = make([]Line, reader.countRead(12))
	for i := range image.ImageLines {
		image.ImageLines[i] = Line{LinePc: reader.uint32Read(), LineFile: reader.uint32Read(), LineNumber: reader.uint32Read()}
	}
	image.ImageFileNames, image.ImageFuncNames = reader.stringsRead(), reader.stringsRead()
	image.ImageLocals = make([]Local, reader.countRead(20))
	for i := range image.ImageLocals {
		local := &image.ImageLocals[i]
		local.LocalOffset, local.LocalSize = reader.uint32Read(), reader.uint32Read()
		local.LocalName, local.LocalTypeName = reader.uint32Read(), reader.uint32Read()
		local.LocalType = types.Code(int32(reader.uint32Read()))
	}
	image.ImageLocalNames = reader.stringsRead()
	image.ImageReadOnly = make([]Span, reader.countRead(8))
	for i := range image.ImageReadOnly {
		image.ImageReadOnly[i] = Span{SpanStart: reader.uint32Read(), SpanEnd: reader.uint32Read()}
	}
	if wordCount := reader.uint32Read(); !reader.imageFileReaderShort && wordCount != constants.OpcodeSetWordCount {
		return Image{}, ImageFileProblemVersion
	}
	for i := range image.ImageOperationsEnabled {
		image.ImageOperationsEnabled[i] = reader.uint64Read()
	}
	if reader.imageFileReaderShort || len(reader.imageFileReaderBytes) != 0 {
		return Image{}, ImageFileProblemTruncated
	}
	return image, ImageFileProblemNone
}
