package constants

// LibraryFunctionDefinitionTable gives every live function of
// LibraryFunctionLifecycleTable its handler and signature, by library and
// then number. t2-cx-core-opcodegen reads it and writes the signature
// tables of libraries_signatures_gen.go and one switch per library, so the
// library functions are fixed when cx is compiled (decision D96). Only the
// process library is left here, as the functions of the file, net and
// cipher libraries are opcodes (decision D119).
//
// Handler is the Go function the switch of the old loop calls,
// package.Function, with the package cxos (t5-cxos, left out by the nocxos
// tag). Inputs and Outputs are spelled as in OpcodeDefinitionTable. Read
// only.
var LibraryFunctionDefinitionTable = [...]LibraryFunctionDefinition{
	// library, number, name, handler, inputs, outputs
	{LibraryNumberProcess, 1, "os.Run", "cxos.OpOsRun", "str i32 i32 str", "i32 i32 str"},
}

// LibraryFunctionDefinition is one row of LibraryFunctionDefinitionTable.
type LibraryFunctionDefinition struct {
	LibraryNumber int
	Number        int
	Name          string
	Handler       string
	Inputs        string
	Outputs       string
}
