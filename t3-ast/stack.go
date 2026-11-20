package ast

import (
	"fmt"

	"github.com/skycoin/skycoin-cx/t0-types"
)

func stackValueHeader(fileName string, fileLine int) string {
	return fmt.Sprintf("%s:%d", fileName, fileLine)
}

// PrintStack ...
func (cxprogram *CXProgram) PrintStack() {
	fmt.Println()
	fmt.Println("===Callstack===")

	// we're going backwards in the stack
	fp := cxprogram.Stack.Pointer

	for c := cxprogram.CallCounter; c != types.InvalidPointer; c-- {
		op := cxprogram.CallStack[c].Operator
		fp -= op.Size

		var dupNames []string

		fmt.Printf(">>> %s()\n", op.Name)

		opInputs := op.GetInputs(cxprogram)
		for _, inputIdx := range opInputs {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S0465.
			input := cxprogram.GetCXTypeSignatureFromArray(inputIdx)

			fmt.Println("ProgramInput")
			fmt.Printf("\t%s : %s() : %s\n", stackValueHeader(CXArgumentDebugFileLine(cxprogram, input.ArgDetails)), op.Name, GetPrintableValue(cxprogram, fp, input))

			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S0466.
			inpPkg, err := cxprogram.GetPackageFromArray(input.Package)
			if err != nil {
				panic(err)
			}
			dupNames = append(dupNames, inpPkg.Name+input.Name)
		}

		opOutputs := op.GetOutputs(cxprogram)
		for _, outputIdx := range opOutputs {
			// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
			// Now an arena getter, site S0467.
			output := cxprogram.GetCXTypeSignatureFromArray(outputIdx)

			fmt.Println("ProgramOutput")
			fmt.Printf("\t%s : %s() : %s\n", stackValueHeader(CXArgumentDebugFileLine(cxprogram, output.ArgDetails)), op.Name, GetPrintableValue(cxprogram, fp, output))

			// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
			// Now an arena getter, site S0468.
			outPkg, err := cxprogram.GetPackageFromArray(output.Package)
			if err != nil {
				panic(err)
			}
			dupNames = append(dupNames, outPkg.Name+output.Name)
		}

		// fmt.Println("Expressions")
		exprs := ""
		for _, expr := range op.Expressions {
			// OBJMODEL: state.CXAtomicOperatorMng.CXAtomicOperatorGet(gid), phase 5 #342.
			// Now an arena getter, site S0469.
			cxAtomicOp, err := cxprogram.GetCXAtomicOp(expr.Index)
			if err != nil {
				panic(err)
			}

			// OBJMODEL: state.CXFunctionMng.CXFunctionGet(gid), phase 5 #352.
			// Now an arena getter, site S0470.
			cxAtomicOpOperator := cxprogram.GetFunctionFromArray(cxAtomicOp.Operator)
			for _, inputIdx := range cxAtomicOp.GetInputs(cxprogram) {
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S0471.
				input := cxprogram.GetCXTypeSignatureFromArray(inputIdx)

				if input.Name == "" || cxAtomicOpOperator == nil {
					continue
				}

				// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
				// Now an arena getter, site S0472.
				inpPkg, err := cxprogram.GetPackageFromArray(input.Package)
				if err != nil {
					panic(err)
				}
				var dup bool
				for _, name := range dupNames {
					if name == inpPkg.Name+input.Name {
						dup = true
						break
					}
				}
				if dup {
					continue
				}

				// fmt.Println("\t", inp.Name, "\t", ":", "\t", GetPrintableValue(fp, inp))
				// exprs += fmt.Sprintln("\t", stackValueHeader(inp.FileName, inp.FileLine), "\t", ":", "\t", GetPrintableValue(fp, inp))
				exprs += fmt.Sprintf("\t%s : %s() : %s\n", stackValueHeader(CXArgumentDebugFileLine(cxprogram, input.ArgDetails)), cxAtomicOp.GetOperatorName(cxprogram), GetPrintableValue(cxprogram, fp, input))

				dupNames = append(dupNames, inpPkg.Name+input.Name)
			}

			for _, outputIdx := range cxAtomicOp.GetOutputs(cxprogram) {
				// OBJMODEL: state.CXTypeSignatureMng.CXTypeSignatureGet(gid), phase 5 #332.
				// Now an arena getter, site S0473.
				output := cxprogram.GetCXTypeSignatureFromArray(outputIdx)

				if output.Type == TYPE_CXARGUMENT_DEPRECATE {
					if output.Name == "" || cxAtomicOpOperator == nil {
						continue
					}

					// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
					// Now an arena getter, site S0474.
					outPkg, err := cxprogram.GetPackageFromArray(output.Package)
					if err != nil {
						panic(err)
					}
					var dup bool
					for _, name := range dupNames {
						if name == outPkg.Name+output.Name {
							dup = true
							break
						}
					}
					if dup {
						continue
					}

					// fmt.Println("\t", out.Name, "\t", ":", "\t", GetPrintableValue(fp, out))
					// exprs += fmt.Sprintln("\t", stackValueHeader(out.FileName, out.FileLine), ":", GetPrintableValue(fp, out))

					exprs += fmt.Sprintf("\t%s : %s() : %s\n", stackValueHeader(CXArgumentDebugFileLine(cxprogram, output.ArgDetails)), cxAtomicOp.GetOperatorName(cxprogram), GetPrintableValue(cxprogram, fp, output))

					dupNames = append(dupNames, outPkg.Name+output.Name)
				} else if output.Type == TYPE_ATOMIC || output.Type == TYPE_POINTER_ATOMIC || output.Type == TYPE_ARRAY_ATOMIC || output.Type == TYPE_POINTER_ARRAY_ATOMIC || output.Type == TYPE_SLICE_ATOMIC || output.Type == TYPE_POINTER_SLICE_ATOMIC {
					if output.Name == "" || cxAtomicOpOperator == nil {
						continue
					}

					// OBJMODEL: state.CXPackageMng.CXPackageGet(gid), phase 5 #362.
					// Now an arena getter, site S0475.
					outputPkg, err := cxprogram.GetPackageFromArray(output.Package)
					if err != nil {
						panic(err)
					}

					var dup bool
					for _, name := range dupNames {
						if name == outputPkg.Name+output.Name {
							dup = true
							break
						}
					}
					if dup {
						continue
					}

					// TODO: Make GetPrintableValue() and other functions receive CXTypeSignature instead of CXArgument
					exprs += fmt.Sprintf("\t%s : %s() : %s\n", stackValueHeader(CXArgumentDebugFileLine(cxprogram, output.ArgDetails)), cxAtomicOp.GetOperatorName(cxprogram), GetPrintableValue(cxprogram, fp, output))

					dupNames = append(dupNames, outputPkg.Name+output.Name)

				}

			}
		}

		if len(exprs) > 0 {
			fmt.Println("Expressions\n", exprs)
		}
	}
}
