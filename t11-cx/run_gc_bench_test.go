package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t1-constants"
	"github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t4-actions"
	"github.com/skycoin/skycoin-cx/t4-execute"
)

// gcBenchPrograms are the GC tests BenchmarkRunGC runs as cxtest does, with
// -heap-initial 0, so nearly every allocation collects.
var gcBenchPrograms = []string{
	"test-escape-gc.cx",
	"test-gc-array-pointer.cx",
	"test-gc-array-field.cx",
	"test-gc-slice-struct.cx",
	"test-gc-nested.cx",
}

// BenchmarkRunGC times execute.RunCompiled on gcBenchPrograms with an
// initial heap of 0 bytes. The compile and RunPrepare, which allocates the
// heap and the collector scratch for the heap cap, are not timed.
func BenchmarkRunGC(b *testing.B) {
	initProgram()
	initHeap := constants.INIT_HEAP_SIZE
	constants.INIT_HEAP_SIZE = 0
	defer func() { constants.INIT_HEAP_SIZE = initHeap }()
	for _, programPath := range gcBenchPrograms {
		source, err := os.ReadFile(filepath.Join("..", "tests", programPath))
		if err != nil {
			b.Fatal(err)
		}
		p := runBenchProgram{name: programPath, source: string(source)}
		b.Run(programPath, func(b *testing.B) {
			discardStdout(b)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				ast.ProgramMemoryRelease(actions.AST)
				compileRunProgram(b, p)
				execute.RunPrepare(actions.AST)
				b.StartTimer()
				if err := execute.RunCompiled(actions.AST, 0, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
