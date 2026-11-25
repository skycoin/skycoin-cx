package constants

// LibraryFunctionDefinitionTable gives every live function of
// LibraryFunctionLifecycleTable its handler, vm form and signature, by
// library and then number. t2-cx-core-opcodegen reads it and writes the
// signature tables of libraries_signatures_gen.go and one switch per
// library, so the library functions are fixed when cx is compiled (decision
// D96).
//
// Handler is the Go function the switch of the old loop calls,
// package.Function, with the package cxos (t5-cxos, the file and process
// libraries, left out by the nocxos tag), cxnet (t4-cxnet) or cipher
// (t4-cipher, left out by nocipher). HandlerVm is the vm form in the same
// package, "" for a function without one. Inputs and Outputs are spelled as
// in OpcodeDefinitionTable, and und is a byte buffer, a ui8 array of any
// length. Read only.
var LibraryFunctionDefinitionTable = [...]LibraryFunctionDefinition{
	// library, number, name, handler, vm handler, inputs, outputs
	{LibraryNumberProcess, 1, "os.Run", "cxos.OpOsRun", "", "str i32 i32 str", "i32 i32 str"},
}

// LibraryFunctionDefinition is one row of LibraryFunctionDefinitionTable.
type LibraryFunctionDefinition struct {
	LibraryNumber int
	Number        int
	Name          string
	Handler       string
	HandlerVm     string
	Inputs        string
	Outputs       string
}
