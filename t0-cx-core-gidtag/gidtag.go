// Package gidtag is the one table of the Gid type tags, see
// docs/GidVariants.md. A tag is fixed forever, like an opcode number, and a
// removed handle keeps its tag. It imports nothing, so ir can use it without
// fmt (docs/Microcontrollers.md).
package gidtag

//go:generate go run ../t1-cx-core-gidtaggen -root ..

// Variant is the bit layout of a Gid. Bits 0 to 31 are the index and 56 to
// 63 the tag in every variant, the variants differ in bits 32 to 55.
type Variant uint8

const (
	// VariantObject has a 24 bit generation that changes on each release.
	VariantObject Variant = 1
	// VariantInterned is never released, its generation stays 1.
	VariantInterned Variant = 2
	// VariantType has the kind in bits 32 to 39 and a 16 bit generation in
	// 40 to 55, see decision D153.
	VariantType Variant = 3
	// VariantLibraryResource has the object layout. A system library
	// releases its slot when the resource is closed.
	VariantLibraryResource Variant = 4
)

// Range is a block of tags kept for one part of CX.
type Range struct {
	RangeName  string
	RangeFirst uint8
	RangeLast  uint8
}

// The tag ranges of decision D154. Tag 0x00 is GidNil and 0x80 to 0xFF are
// never used, so every valid Gid is positive.
var (
	RangeCompiler      = Range{"compiler and IR objects, interned pools", 0x01, 0x1F}
	RangeSystemLibrary = Range{"system library resources", 0x20, 0x3F}
	RangeReflect       = Range{"sys/reflect, sys/meta, sys/cxir and sys/compiler handles", 0x40, 0x5F}
	RangeReserved      = Range{"reserved", 0x60, 0x7F}
)

// RangeTable lists the ranges in tag order.
var RangeTable = []Range{RangeCompiler, RangeSystemLibrary, RangeReflect, RangeReserved}

// Tag is one row of Table. TagName is also the name of its generated
// constant and its entry in NameTable. TagOwner is the package that makes
// Gids of the tag.
type Tag struct {
	TagName    string
	TagValue   uint8
	TagVariant Variant
	TagRange   Range
	TagOwner   string
}

// Table is every tag in use. Run go generate in this directory, or make
// generate-gidtags, after changing it.
var Table = []Tag{
	{"CXProgram", 0x01, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXPackage", 0x02, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXFunction", 0x03, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXNativeFunction", 0x04, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXStruct", 0x05, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXExpression", 0x06, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXAtomicOperator", 0x07, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXLine", 0x08, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXArgument", 0x09, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXArgumentDebug", 0x0A, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXTypeSignature", 0x0B, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXTypeSignature_Array", 0x0C, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXTypeSignature_Struct", 0x0D, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"CXCall", 0x0E, VariantObject, RangeCompiler, "t3-cx-core-ast"},
	{"String", 0x0F, VariantInterned, RangeCompiler, "t2-cx-core-stringpool"},
	{"SourceFile", 0x10, VariantInterned, RangeCompiler, "t3-cx-core-ast"},
	{"ReflectTypeName", 0x11, VariantInterned, RangeCompiler, "t2-cx-library-reflecttable"},
	{"CxirName", 0x12, VariantInterned, RangeCompiler, "t2-cx-runtime-ir"},
	{"ReflectType", 0x40, VariantType, RangeReflect, "t2-cx-library-reflecttable"},
	{"ReflectFunction", 0x41, VariantInterned, RangeReflect, "t2-cx-library-reflecttable"},
	{"MetaPackage", 0x42, VariantInterned, RangeReflect, "t2-cx-library-reflecttable"},
	{"CxirImage", 0x43, VariantLibraryResource, RangeReflect, "t2-cx-runtime-ir"},
	{"CompilerAst", 0x44, VariantLibraryResource, RangeReflect, "t4-cx-library-compileropcodes"},
}
