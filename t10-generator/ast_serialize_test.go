package generator_test

import (
	"bytes"
	"encoding/binary"
	// "reflect" // only used by the FAILING tests below
	"testing"
	// "time" // only used by the FAILING tests below

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

// FAILING: t3-ast serializeExpression and serializeFunctionArguments drop TYPE_ATOMIC signatures via
// ConvertIndexTypeSignaturesToPointerArgs, and deserializePackages leaves pkg.Globals nil; see #12
// func TestSerialize(t *testing.T) {
// 	tests := []struct {
// 		scenario          string
// 		program           *cxast.CXProgram
// 		includeDataMemory bool
// 		useCompression    bool
// 	}{
// 		{
// 			scenario:          "program without literal",
// 			program:           generateSampleProgramFromCXEvolves(t, false),
// 			includeDataMemory: false,
// 			useCompression:    false,
// 		},
// 		{
// 			scenario:          "program with literal",
// 			program:           generateSampleProgramFromCXEvolves(t, true),
// 			includeDataMemory: true,
// 			useCompression:    false,
// 		},
// 		{
// 			scenario:          "program without literal, use compression",
// 			program:           generateSampleProgramFromCXEvolves(t, false),
// 			includeDataMemory: false,
// 			useCompression:    true,
// 		},
// 		{
// 			scenario:          "program with literal, use compression",
// 			program:           generateSampleProgramFromCXEvolves(t, true),
// 			includeDataMemory: true,
// 			useCompression:    true,
// 		},
// 	}
//
// 	for _, tc := range tests {
// 		t.Run(tc.scenario, func(t *testing.T) {
// 			timeStart := time.Now()
// 			serializedBytes := cxast.SerializeCXProgram(tc.program, tc.includeDataMemory, tc.useCompression)
// 			t.Logf("Serialization took=%v\n", time.Since(timeStart))
//
// 			timeStart = time.Now()
// 			deserializedCXProgram := cxast.Deserialize(serializedBytes, tc.useCompression)
// 			t.Logf("Deserialization took=%v\n", time.Since(timeStart))
//
// 			if cxast.ToString(deserializedCXProgram) != cxast.ToString(tc.program) {
// 				t.Errorf("%+v\n!=\n%+v\n", cxast.ToString(deserializedCXProgram), cxast.ToString(tc.program))
// 			}
//
// 			if tc.includeDataMemory {
// 				wantDataSegmentMemory := tc.program.Memory[tc.program.Data.StartsAt : tc.program.Data.StartsAt+tc.program.Data.Size]
// 				gotDataSegmentMemory := deserializedCXProgram.Memory[deserializedCXProgram.Data.StartsAt : deserializedCXProgram.Data.StartsAt+deserializedCXProgram.Data.Size]
// 				if !reflect.DeepEqual(wantDataSegmentMemory, gotDataSegmentMemory) {
// 					t.Errorf("want %v, got %v", wantDataSegmentMemory, gotDataSegmentMemory)
// 				}
// 			}
// 		})
// 	}
// }

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

	fn, _ := cxProgram.GetFunction("TestFunction", "main")
	pkg, _ := cxProgram.GetPackage("main")
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
