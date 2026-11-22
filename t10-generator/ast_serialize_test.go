package generator_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"regexp"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-lz4codec"
	"github.com/skycoin/skycoin-cx/t0-types"
	cxgenerator "github.com/skycoin/skycoin-cx/t10-generator"
	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	cxinit "github.com/skycoin/skycoin-cx/t7-init"                           // thin: tests use only MakeProgram, RegisterPackages; no depth gain
	cxparsingcompletor "github.com/skycoin/skycoin-cx/t8-cxparsingcompletor" // thin: tests use only InitCXCore; no depth gain
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

// t.Logf("Stack size=%v\n", tc.program.StackSize)
// t.Logf("dataSegment size=%v\n", tc.program.DataSegmentSize)
// t.Logf("dataSegment starts at=%v\n", tc.program.DataSegmentStartsAt)
// t.Logf("heap size=%v\n", tc.program.HeapSize)
// t.Logf("heap starts at=%v\n", tc.program.HeapStartsAt)
// t.Logf("memory len=%v\n", len(tc.program.Memory))
// t.Logf("value at memory data offset=%v\n", tc.program.Memory[tc.program.DataSegmentStartsAt:tc.program.DataSegmentStartsAt+tc.program.DataSegmentSize])

// sampleOperand matches an expression input in ToString, as in i32.add(15 i32, 11 i32).
var sampleOperand = regexp.MustCompile(`\(\w+ (i32|bool)[,)]`)

// TestSerialize round trips the sample program, whose operands are TYPE_ATOMIC
// type signatures, with and without memory and compression. See #12.
func TestSerialize(t *testing.T) {
	tests := []struct {
		scenario          string
		withLiteral       bool
		includeDataMemory bool
		useCompression    bool
	}{
		{"program without literal", false, false, false},
		{"program with literal", true, true, false},
		{"program without literal, use compression", false, false, true},
		{"program with literal, use compression", true, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			program := generateSampleProgramFromCXEvolves(t, tc.withLiteral)
			var codec *lz4codec.Codec
			if tc.useCompression {
				codec = newProgramCodec(t)
			}
			serializedBytes, err := cxast.SerializeCXProgram(program, tc.includeDataMemory, codec)
			if err != nil {
				t.Fatal(err)
			}
			// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
			// Now a constructor call, as the other round trip tests make it.
			deserializedCXProgram, err := cxast.Deserialize(serializedBytes, codec)
			if err != nil {
				t.Fatal(err)
			}

			want, got := cxast.ToString(program), cxast.ToString(deserializedCXProgram)
			if got != want {
				t.Errorf("%+v\n!=\n%+v\n", got, want)
			}
			if !sampleOperand.MatchString(want) {
				t.Errorf("the sample program has no expression with an operand to compare:\n%s", want)
			}

			if tc.includeDataMemory || tc.useCompression {
				wantDataSegmentMemory := program.Memory[program.Data.StartsAt : program.Data.StartsAt+program.Data.Size]
				gotDataSegmentMemory := deserializedCXProgram.Memory[deserializedCXProgram.Data.StartsAt : deserializedCXProgram.Data.StartsAt+deserializedCXProgram.Data.Size]
				if !reflect.DeepEqual(wantDataSegmentMemory, gotDataSegmentMemory) {
					t.Errorf("want %v, got %v", wantDataSegmentMemory, gotDataSegmentMemory)
				}
			}
		})
	}
}

func generateSampleProgramFromCXEvolves(t *testing.T, withLiteral bool) *cxast.CXProgram {
	var cxProgram *cxast.CXProgram

	// Needed for AddNativeExpressionToFunction
	// because of dependency on cxast.OpNames
	cxparsingcompletor.InitCXCore()
	// OBJMODEL: state.CXProgramMng.CXProgramAlloc(), phase 4 #371.
	// Now a constructor call, site S0013.
	cxProgram = cxinit.MakeProgram()
	cxinit.RegisterPackages(cxProgram)

	// OBJMODEL: state.CXPackageMng.CXPackageAlloc(), phase 4 #361.
	// Now an arena insert, site S0014.
	err := astapi.AddEmptyPackage(cxProgram, "main")
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	// OBJMODEL: state.CXFunctionMng.CXFunctionAlloc(), phase 4 #351.
	// Now an arena insert, site S0015.
	err = astapi.AddEmptyFunctionToPackage(cxProgram, "main", "TestFunction")
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	err = astapi.AddNativeInputToFunction(cxProgram, "main", "TestFunction", "inputOne", types.I32)
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}

	err = astapi.AddNativeOutputToFunction(cxProgram, "main", "TestFunction", "outputOne", types.I32)
	if err != nil {
		t.Errorf("want no error, got %v", err)
	}
	functionSetNames := []string{"i32.add", "i32.mul", "i32.sub", "i32.eq", "i32.uneq", "i32.gt", "i32.gteq", "i32.lt", "i32.lteq", "bool.not", "bool.or", "bool.and", "bool.uneq", "bool.eq", "i32.neg", "i32.abs", "i32.bitand", "i32.bitor", "i32.bitxor", "i32.bitclear", "i32.bitshl", "i32.bitshr", "i32.max", "i32.min", "i32.rand"}
	fns := cxgenerator.GetFunctionSet(functionSetNames)

	fn, _ := cxProgram.GetFunction(cxProgram.CXProgramNameIntern("TestFunction"), cxProgram.CxIrState.CxIrStateNameMain)
	pkg, _ := cxProgram.GetPackage(cxProgram.CxIrState.CxIrStateNameMain)
	cxgenerator.GenerateRandomExpressions(cxProgram, fn, pkg, fns, 30)

	if withLiteral {
		buf := new(bytes.Buffer)
		var num int32 = 5
		binary.Write(buf, binary.LittleEndian, num)
		err = astapi.AddLiteralInputToExpression(cxProgram, "main", "TestFunction", buf.Bytes(), types.I32, 2)
		if err != nil {
			t.Errorf("want no error, got %v", err)
		}

	}

	cxProgram.PrintProgram()
	return cxProgram
}
