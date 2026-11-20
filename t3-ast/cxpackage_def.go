package ast

// CXPackage is used to represent a CX package.
type CXPackage struct {
	// Metadata
	Name  string // Name of the package
	Index int    // Index of package inside the CXPackage array

	// Contents
	Imports   map[string]CXPackageIndex  // imported packages
	Functions map[string]CXFunctionIndex // declared functions in this package
	Structs   map[string]CXStructIndex   // declared structs in this package
	// OBJMODEL: CXStructGid, phase 5 #337.
	// Now a pointer at rest, site S0157.
	Globals *CXStruct // declared global variables in this package

	// Used by the REPL and cxgo
	CurrentFunction CXFunctionIndex
}
