package ast

// CXStruct is used to represent a CX struct.
type CXStruct struct {
	// Metadata
	Index   int
	Name    string         // Name of the struct
	Package CXPackageIndex // The package this struct belongs to

	// Contents
	Fields []CXTypeSignatureIndex // The fields of the struct
}
