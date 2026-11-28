package opcodetable

import (
	"github.com/skycoin/skycoin-cx/t0-cx-core-valuetypes" // thin: only PointerSize, Pointer; core types, keep
)

// var COREPATH = ""

const StackOverflowError = "stack overflow"
const MainFunction = "main"
const SysInitFunction = "*init"
const MainPackage = "main"
const StdlibPackage = "stdlib"
const OsPackage = "os"
const OsArgs = "Args"

// LanguageVersionCurrent is the language version this CX implements, a
// yymmddnnnn integer stored in serialized programs as CXProgram.Version.
// See the Language versions section of LanguageGuide.md.
const LanguageVersionCurrent int64 = 2611280001

const NonAssignPrefix = "nonAssign"
const LocalPrefix = "*tmp"

// SliceLiteralPrefix names the slice a slice literal appends its elements
// to. It starts with LocalPrefix, so the slice is a temp.
const SliceLiteralPrefix = LocalPrefix + "slc"
const LabelPrefix = "*lbl"

// ResultName names the result of func f() T, which has no name in the source.
const ResultName = "*result"

// ElementPrefix names the variable that holds a struct literal element of a
// slice literal. It is not a temp, so its field assignments are typed normally.
const ElementPrefix = "*elt"

// ArrayElementPrefix names the element variable of an array literal. It starts
// with ElementPrefix, so the element is typed as a slice literal element.
const ArrayElementPrefix = ElementPrefix + "arr"

// MethodResultPrefix names the variable that holds the result of a method
// call used as a value. It is not a temp, so a read of it is looked up.
const MethodResultPrefix = "*mrs"

// MultipleValuePrefix names the variable that holds one value of a, b = x, y
// until every value is computed.
const MultipleValuePrefix = "*mva"

// Used in `ProgramPrint` to represent literals (`CXArgument`s with no name).
const LiteralPlaceholder = "*lit"
const IDFunction = "identity"
const InitFunction = "initDef"

const CallStackSize valuetypes.Pointer = 1000

var StackSize valuetypes.Pointer = 1048576 // 1 Mb
// VIOLATION: state outside State. Owner: t1-cx-core-opcodetable; moves to ConfigMng in phase 2.
var HeapSizeInit valuetypes.Pointer = 2097152 // 2 Mb
// VIOLATION: state outside State. Owner: t1-cx-core-opcodetable; moves to ConfigMng in phase 2.
var MaxHeapSize valuetypes.Pointer = 67108864 // 64 Mb
// VIOLATION: state outside State. Owner: t1-cx-core-opcodetable; moves to ConfigMng in phase 2.
var MinHeapFreeRatio float32 = 0.4

// VIOLATION: state outside State. Owner: t1-cx-core-opcodetable; moves to ConfigMng in phase 2.
var MaxHeapFreeRatio float32 = 0.7

const NullHeapAddressOffset = valuetypes.PointerSize

// EscapeOffsetFlag marks the offset of a local that lives in a heap object.
// Memory stays below it, see ast.OffsetEscaped.
const EscapeOffsetFlag valuetypes.Pointer = 1 << 31
const NullHeapAddress = valuetypes.Pointer(0)
const SliceHeaderSize = 2 * valuetypes.PointerSize

//VERY WEIRD
//gives error, "cx" not found, even if it exists when changed

/*
grep -rn "PARAM_DEFAULT" .
./cx/config.go:87:	ParamDefault = iota
./cx/config.go:96:	ParamDefault
./cx/opcodes.go:843:	case ParamDefault:

grep -rn "PARAM_SLICE" .
./cx/config.go:88:	ParamSlice
./cx/config.go:97:	ParamSlice
./cx/opcodes.go:845:	case ParamSlice:

grep -rn "PARAM_STRUCT" .
./cx/config.go:89:	ParamStruct
./cx/config.go:98:	ParamStruct
./cx/opcodes.go:847:	case ParamStruct:

*/

// works
// BUG
const (
	ParamDefault = iota
	ParamSlice
	ParamStruct
)

/*
//doesnt work
const (
	PARAM_UNUSED = iota
	ParamDefault
	ParamSlice
	ParamStruct
)
*/

var ErrorStrings map[int]string = map[int]string{
	valuetypes.CXSuccess:                     "CX_SUCCESS",
	valuetypes.CXCompilationError:            "CX_COMPILATION_ERROR",
	valuetypes.CXPanic:                       "CX_PANIC",
	valuetypes.CXInternalError:               "CX_INTERNAL_ERROR",
	valuetypes.CXAssert:                      "CX_ASSERT",
	valuetypes.CXRuntimeError:                "CX_RUNTIME_ERROR",
	valuetypes.CXRuntimeStackOverflowError:   "CX_RUNTIME_STACK_OVERFLOW_ERROR",
	valuetypes.CXRuntimeHeapExhaustedError:   "CX_RUNTIME_HEAP_EXHAUSTED_ERROR",
	valuetypes.CXRuntimeInvalidArgument:      "CX_RUNTIME_INVALID_ARGUMENT",
	valuetypes.CXRuntimeSliceIndexOutOfRange: "CX_RUNTIME_SLICE_INDEX_OUT_OF_RANGE",
	valuetypes.CXRuntimeNotImplemented:       "CX_RUNTIME_NOT_IMPLEMENTED",
	valuetypes.CXRuntimeInvalidCast:          "CX_RUNTIME_INVALID_CAST",
	valuetypes.CXRuntimeLibraryDisabled:      "CX_RUNTIME_LIBRARY_DISABLED",
}

const (
	DeclarationUnused   = iota //if this value appears, program should crash; is error
	DeclarationPointer         // 1
	DeclarationDeref           // 2
	DeclarationArray           // 3
	DeclarationSlice           // 4
	DeclarationStruct          // 5
	DeclarationIndexing        // 6
	DeclarationBasic           // 7
	DeclarationFunc            // 8
)

// ERROR: see below,
const (
	//PASSBY_UNUSED = iota
	PassByValue = iota
	PassByReference
)

const (
	DerefUnused  = iota //reserve zero value, if this value appears, program should crash; should be assert
	DerefArray          // 1
	DerefField          // 2
	DerefPointer        // 3
	DerefDeref          // 4
	DerefSlice          // 5
)
