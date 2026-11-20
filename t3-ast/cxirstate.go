package ast

//go:generate go run ../t0-cx-core-objmodelgen -dir .

import (
	constants "github.com/skycoin/skycoin-cx/t1-constants"
	objmodel "github.com/skycoin/skycoin-cx/t1-cx-core-objmodel"
)

// CxIrState holds one manager of each IR object type and is the only owner of
// every IR object, see docs/ObjectModel.md. Nothing uses it yet, #299 stage 4
// moves the allocation sites onto it.
type CxIrState struct {
	CXProgramMng              CXProgramMng
	CXPackageMng              CXPackageMng
	CXFunctionMng             CXFunctionMng
	CXNativeFunctionMng       CXNativeFunctionMng
	CXStructMng               CXStructMng
	CXExpressionMng           CXExpressionMng
	CXAtomicOperatorMng       CXAtomicOperatorMng
	CXLineMng                 CXLineMng
	CXArgumentMng             CXArgumentMng
	CXArgumentDebugMng        CXArgumentDebugMng
	CXTypeSignatureMng        CXTypeSignatureMng
	CXTypeSignature_ArrayMng  CXTypeSignature_ArrayMng
	CXTypeSignature_StructMng CXTypeSignature_StructMng
	CXCallMng                 CXCallMng
}

// CxIrStateManagerLen is one manager's name and Len, for the teardown test.
type CxIrStateManagerLen struct {
	Name string
	Len  int
}

// CxIrStateInit sets up every manager. One state holds one program, and the
// call stack has its fixed size. The other managers start empty and grow,
// as the arenas they replace do.
func (state *CxIrState) CxIrStateInit() {
	state.CXProgramMng.CXProgramMngInit(1, false)
	state.CXPackageMng.CXPackageMngInit(0, true)
	state.CXFunctionMng.CXFunctionMngInit(0, true)
	state.CXNativeFunctionMng.CXNativeFunctionMngInit(0, true)
	state.CXStructMng.CXStructMngInit(0, true)
	state.CXExpressionMng.CXExpressionMngInit(0, true)
	state.CXAtomicOperatorMng.CXAtomicOperatorMngInit(0, true)
	state.CXLineMng.CXLineMngInit(0, true)
	state.CXArgumentMng.CXArgumentMngInit(0, true)
	state.CXArgumentDebugMng.CXArgumentDebugMngInit(0, true)
	state.CXTypeSignatureMng.CXTypeSignatureMngInit(0, true)
	state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayMngInit(0, true)
	state.CXTypeSignature_StructMng.CXTypeSignature_StructMngInit(0, true)
	state.CXCallMng.CXCallMngInit(int(constants.CALLSTACK_SIZE), false)
}

// CxIrStateTeardown1 releases the program. Releasing every object the
// program owns through the owner relations is #300, once they are Gids.
func (state *CxIrState) CxIrStateTeardown1() {
	manager := &state.CXProgramMng
	for index, live := range manager.CXProgramMngLive {
		if live {
			generation := manager.CXProgramMngGenerations[index]
			manager.CXProgramRelease(CXProgramGid(objmodel.GidCompose(CXProgramGidTypeTag, generation, uint32(index))))
		}
	}
}

// CxIrStateTeardown2 tears every manager down, which asserts that its Len is 0.
func (state *CxIrState) CxIrStateTeardown2() {
	state.CXProgramMng.CXProgramMngTeardown()
	state.CXPackageMng.CXPackageMngTeardown()
	state.CXFunctionMng.CXFunctionMngTeardown()
	state.CXNativeFunctionMng.CXNativeFunctionMngTeardown()
	state.CXStructMng.CXStructMngTeardown()
	state.CXExpressionMng.CXExpressionMngTeardown()
	state.CXAtomicOperatorMng.CXAtomicOperatorMngTeardown()
	state.CXLineMng.CXLineMngTeardown()
	state.CXArgumentMng.CXArgumentMngTeardown()
	state.CXArgumentDebugMng.CXArgumentDebugMngTeardown()
	state.CXTypeSignatureMng.CXTypeSignatureMngTeardown()
	state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayMngTeardown()
	state.CXTypeSignature_StructMng.CXTypeSignature_StructMngTeardown()
	state.CXCallMng.CXCallMngTeardown()
}

// CxIrStateManagerLens returns each manager's name and Len.
func (state *CxIrState) CxIrStateManagerLens() []CxIrStateManagerLen {
	return []CxIrStateManagerLen{
		{"CXProgramMng", state.CXProgramMng.CXProgramLen()},
		{"CXPackageMng", state.CXPackageMng.CXPackageLen()},
		{"CXFunctionMng", state.CXFunctionMng.CXFunctionLen()},
		{"CXNativeFunctionMng", state.CXNativeFunctionMng.CXNativeFunctionLen()},
		{"CXStructMng", state.CXStructMng.CXStructLen()},
		{"CXExpressionMng", state.CXExpressionMng.CXExpressionLen()},
		{"CXAtomicOperatorMng", state.CXAtomicOperatorMng.CXAtomicOperatorLen()},
		{"CXLineMng", state.CXLineMng.CXLineLen()},
		{"CXArgumentMng", state.CXArgumentMng.CXArgumentLen()},
		{"CXArgumentDebugMng", state.CXArgumentDebugMng.CXArgumentDebugLen()},
		{"CXTypeSignatureMng", state.CXTypeSignatureMng.CXTypeSignatureLen()},
		{"CXTypeSignature_ArrayMng", state.CXTypeSignature_ArrayMng.CXTypeSignature_ArrayLen()},
		{"CXTypeSignature_StructMng", state.CXTypeSignature_StructMng.CXTypeSignature_StructLen()},
		{"CXCallMng", state.CXCallMng.CXCallLen()},
	}
}
