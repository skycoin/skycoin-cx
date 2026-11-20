package ast

type CXAtomicOperator struct {
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0087.
	Inputs *CXStruct
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0088.
	Outputs  *CXStruct
	Operator CXFunctionIndex

	Function CXFunctionIndex
	Package  CXPackageIndex
	Label    string

	// used for jmp statements
	ThenLines int
	ElseLines int
}
