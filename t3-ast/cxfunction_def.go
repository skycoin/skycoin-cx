package ast

import "github.com/skycoin/skycoin-cx/t0-types"

// CXFunction is used to represent a CX function.
type CXFunction struct {
	// Metadata
	Index        int
	Name         string         // Name of the function
	Package      CXPackageIndex // The package it's a member of
	AtomicOPCode int

	// Contents
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0127.
	Inputs *CXStruct // Input parameters to the function
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0128.
	Outputs *CXStruct // Output parameters from the function
	// OBJMODEL: []CXExpressionGid or CXExpressionGid, phase 5 #347.
	// Now objects held by value, site S0129.
	Expressions []CXExpression // Expressions, including control flow statements, in the function

	// contains the name of its local variables
	// Used to determine if the variables are to be added to the symbols array
	// Used for local scoping
	LocalVariables []string

	//TODO: Better Comment for this
	LineCount int // number of expressions, pre-computed for performance

	//TODO: Better Comment for this
	Size types.Pointer // automatic memory size

	// Debugging
	FileName string
	FileLine int

	// Used by the GC
	// OBJMODEL: CXArgumentGid, phase 5 #327.
	// Now a pointer at rest, site S0130.
	ListOfPointers []*CXArgument // Root pointers for the GC algorithm
}
