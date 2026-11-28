package ir

import (
	"encoding/binary"

	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes"
	"github.com/skycoin/skycoin-cx/t0-cx-runtime-output"
	"github.com/skycoin/skycoin-cx/t1-cx-core-opcodetable"
)

// ImageFileMagic starts every image file, see docs/Microcontrollers.md.
const ImageFileMagic = "CXIMAGE\x00"

// ImageFileVersion is the layout version of an image file. Opcode numbers
// change from one cx version to the next, so a file also stores OpcodeCount
// and InstrSize, and ImageFileDecode refuses a file whose values differ.
const ImageFileVersion = 3

// ImageFileEncode returns image as bytes that ImageFileDecode reads back,
// every field little endian in declaration order.
func ImageFileEncode(image *Image) []byte {
	file := make([]byte, ImageFileEncodedSize(image))
	ImageFileEncodeInto(file, image)
	return file
}

// ImageFileEncodedSize returns the size of the file ImageFileEncode writes.
func ImageFileEncodedSize(image *Image) int {
	writer := imageFileWriter{}
	writer.imageWrite(image)
	return writer.imageFileWriterAt
}

// ImageFileEncodeInto writes the file of image into file, which holds at
// least ImageFileEncodedSize bytes, and returns its size, -1 when file is
// shorter. It allocates nothing, so sys/cxir encodes in the run (D153).
func ImageFileEncodeInto(file []byte, image *Image) int {
	size := ImageFileEncodedSize(image)
	if len(file) < size {
		return -1
	}
	writer := imageFileWriter{imageFileWriterBytes: file}
	writer.imageWrite(image)
	return size
}

// imageFileWriter writes the fields of an image file in order. Without
// bytes it only counts them.
type imageFileWriter struct {
	imageFileWriterBytes []byte
	imageFileWriterAt    int
}

func (writer *imageFileWriter) uint16Write(value uint16) {
	if writer.imageFileWriterBytes != nil {
		binary.LittleEndian.PutUint16(writer.imageFileWriterBytes[writer.imageFileWriterAt:], value)
	}
	writer.imageFileWriterAt += 2
}

func (writer *imageFileWriter) uint32Write(value uint32) {
	if writer.imageFileWriterBytes != nil {
		binary.LittleEndian.PutUint32(writer.imageFileWriterBytes[writer.imageFileWriterAt:], value)
	}
	writer.imageFileWriterAt += 4
}

func (writer *imageFileWriter) uint64Write(value uint64) {
	if writer.imageFileWriterBytes != nil {
		binary.LittleEndian.PutUint64(writer.imageFileWriterBytes[writer.imageFileWriterAt:], value)
	}
	writer.imageFileWriterAt += 8
}

func (writer *imageFileWriter) bytesWrite(bytes []byte) {
	writer.uint32Write(uint32(len(bytes)))
	if writer.imageFileWriterBytes != nil {
		copy(writer.imageFileWriterBytes[writer.imageFileWriterAt:], bytes)
	}
	writer.imageFileWriterAt += len(bytes)
}

func (writer *imageFileWriter) stringsWrite(strs []string) {
	writer.uint32Write(uint32(len(strs)))
	for _, str := range strs {
		writer.uint32Write(uint32(len(str)))
		if writer.imageFileWriterBytes != nil {
			copy(writer.imageFileWriterBytes[writer.imageFileWriterAt:], str)
		}
		writer.imageFileWriterAt += len(str)
	}
}

func (writer *imageFileWriter) slotsWrite(slots []Slot) {
	writer.uint32Write(uint32(len(slots)))
	for _, slot := range slots {
		writer.uint32Write(slot.SlotOffset)
		writer.uint32Write(slot.SlotSize)
	}
}

// imageFileMagicBytes is ImageFileMagic as bytes, which the run path
// compares and writes without a string.
var imageFileMagicBytes = [...]byte{'C', 'X', 'I', 'M', 'A', 'G', 'E', 0}

// imageWrite writes every field of image.
func (writer *imageFileWriter) imageWrite(image *Image) {
	writer.imageWriteHead(image)
	writer.stringsWrite(image.ImageFileNames)
	writer.stringsWrite(image.ImageFuncNames)
	writer.imageWriteLocals(image)
	writer.stringsWrite(image.ImageLocalNames)
	writer.imageWriteTail(image)
}

// imageWriteHead writes the fields before the file names. With
// imageWriteLocals and imageWriteTail it writes every field but the name
// tables, so ImageStore writes those from its names without a string.
func (writer *imageFileWriter) imageWriteHead(image *Image) {
	if writer.imageFileWriterBytes != nil {
		copy(writer.imageFileWriterBytes, imageFileMagicBytes[:])
	}
	writer.imageFileWriterAt += len(imageFileMagicBytes)
	writer.uint32Write(ImageFileVersion)
	writer.uint32Write(uint32(OpcodeCount))
	writer.uint32Write(InstrSize)
	writer.uint32Write(uint32(len(image.ImageFuncs)))
	for i := range image.ImageFuncs {
		fn := &image.ImageFuncs[i]
		writer.uint32Write(uint32(len(fn.FuncCode)))
		for j := range fn.FuncCode {
			in := &fn.FuncCode[j]
			writer.uint16Write(uint16(in.InstrOpcode))
			writer.uint16Write(uint16(in.InstrMode))
			for _, operand := range [...]uint32{in.InstrA, in.InstrB, in.InstrC, in.InstrD, in.InstrN, in.InstrM, in.InstrAux} {
				writer.uint32Write(operand)
			}
		}
		writer.uint32Write(fn.FuncFrameSize)
		writer.slotsWrite(fn.FuncInputs)
		writer.slotsWrite(fn.FuncOutputs)
		for _, value := range [...]uint32{fn.FuncLineFirst, fn.FuncLineCount, fn.FuncLocalFirst, fn.FuncLocalCount} {
			writer.uint32Write(value)
		}
	}
	writer.bytesWrite(image.ImageData)
	for _, value := range [...]uint32{image.ImageDataStart, image.ImageStackSize, image.ImageScratchSize} {
		writer.uint32Write(value)
	}
	writer.bytesWrite(image.ImageText)
	writer.uint32Write(image.ImageEntryInit)
	writer.uint32Write(image.ImageEntryMain)
	writer.uint32Write(uint32(len(image.ImageLines)))
	for _, line := range image.ImageLines {
		writer.uint32Write(line.LinePc)
		writer.uint32Write(line.LineFile)
		writer.uint32Write(line.LineNumber)
	}
}

// imageWriteLocals writes the variables, after the function names.
func (writer *imageFileWriter) imageWriteLocals(image *Image) {
	writer.uint32Write(uint32(len(image.ImageLocals)))
	for _, local := range image.ImageLocals {
		writer.uint32Write(local.LocalOffset)
		writer.uint32Write(local.LocalSize)
		writer.uint32Write(local.LocalName)
		writer.uint32Write(local.LocalTypeName)
		writer.uint32Write(uint32(int32(local.LocalType)))
	}
}

// imageWriteTail writes the fields after the variable names.
func (writer *imageFileWriter) imageWriteTail(image *Image) {
	writer.uint32Write(uint32(len(image.ImageReadOnly)))
	for _, span := range image.ImageReadOnly {
		writer.uint32Write(span.SpanStart)
		writer.uint32Write(span.SpanEnd)
	}
	writer.uint32Write(opcodetable.OpcodeSetWordCount)
	for _, word := range image.ImageOperationsEnabled {
		writer.uint64Write(word)
	}
	writer.uint32Write(uint32(image.ImageOutputMode))
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
		local.LocalType = valuetypes.Code(int32(reader.uint32Read()))
	}
	image.ImageLocalNames = reader.stringsRead()
	image.ImageReadOnly = make([]Span, reader.countRead(8))
	for i := range image.ImageReadOnly {
		image.ImageReadOnly[i] = Span{SpanStart: reader.uint32Read(), SpanEnd: reader.uint32Read()}
	}
	if wordCount := reader.uint32Read(); !reader.imageFileReaderShort && wordCount != opcodetable.OpcodeSetWordCount {
		return Image{}, ImageFileProblemVersion
	}
	for i := range image.ImageOperationsEnabled {
		image.ImageOperationsEnabled[i] = reader.uint64Read()
	}
	if mode := reader.uint32Read(); mode < uint32(output.ModeCount) {
		image.ImageOutputMode = output.Mode(mode)
	} else if !reader.imageFileReaderShort {
		return Image{}, ImageFileProblemVersion
	}
	if reader.imageFileReaderShort || len(reader.imageFileReaderBytes) != 0 {
		return Image{}, ImageFileProblemTruncated
	}
	return image, ImageFileProblemNone
}
