package ast

import (
	"testing"

	stringpool "github.com/skycoin/skycoin-cx/t2-cx-core-stringpool"
)

func TestCXPackageMethods(t *testing.T) {
	prgrm := &CXProgram{
		Packages:       make(map[stringpool.StringGid]CXPackageIndex),
		CurrentPackage: -1,
		CxIrState:      &CxIrState{},
	}
	prgrm.CxIrState.CxIrStateInit()
	pkgIdx := prgrm.AddPackage(MakePackage(prgrm.CXProgramNameIntern("geometry")))
	pkg, err := prgrm.GetPackageFromArray(pkgIdx)
	if err != nil {
		t.Fatal(err)
	}

	point := prgrm.CXProgramNameIntern("Point")
	dist := prgrm.CXProgramNameIntern("dist")
	fullName := prgrm.CXProgramMethodNameIntern(point, dist)
	if got := prgrm.CXProgramNameText(fullName); got != "Point.dist" {
		t.Fatalf("method name %q, want Point.dist", got)
	}
	_, fnIdx := pkg.AddMethod(prgrm, methodFunction(prgrm, fullName), point, dist)

	if got, err := pkg.GetMethod(prgrm, point, dist); err != nil || got != fnIdx {
		t.Errorf("GetMethod(Point, dist) = %d, %v, want %d", got, err, fnIdx)
	}
	if got, ok := pkg.Functions[fullName]; !ok || got != fnIdx {
		t.Errorf("Functions[Point.dist] = %d, %v, want %d", got, ok, fnIdx)
	}
	if _, err := pkg.GetMethod(prgrm, dist, point); err == nil {
		t.Error("GetMethod(dist, Point) found a method")
	}

	pkg.RemoveFunction(prgrm, fullName)
	if _, err := pkg.GetMethod(prgrm, point, dist); err == nil {
		t.Error("GetMethod found a removed method")
	}
}

func methodFunction(prgrm *CXProgram, name stringpool.StringGid) *CXFunction {
	_, fn := MakeFunction(prgrm, name, SourceFileGidNil, 0)
	return fn
}
