package generator_test

import (
	"bytes"
	"reflect"
	"sort"
	"testing"

	cxast "github.com/skycoin/skycoin-cx/t3-ast"
	"github.com/skycoin/skycoin-cx/t9-astapi"
)

// sampleProgramWithPackages is the sample program plus packages that each
// declare functions, so the serializer has several packages to order.
func sampleProgramWithPackages(t *testing.T) *cxast.CXProgram {
	t.Helper()
	prgrm := generateSampleProgramFromCXEvolves(t, false)
	for _, pkg := range []string{"alpha", "beta", "gamma", "delta"} {
		if err := astapi.AddEmptyPackage(prgrm, pkg); err != nil {
			t.Fatal(err)
		}
		for _, fn := range []string{"one", "two", "three"} {
			if err := astapi.AddEmptyFunctionToPackage(prgrm, pkg, pkg+"_"+fn); err != nil {
				t.Fatal(err)
			}
		}
	}
	return prgrm
}

// packageFunctions returns the function names of each package.
func packageFunctions(prgrm *cxast.CXProgram) map[string][]string {
	got := map[string][]string{}
	for name, pkgIdx := range prgrm.Packages {
		pkg, err := prgrm.GetPackageFromArray(pkgIdx)
		if err != nil {
			continue
		}
		fns := []string{}
		for fnName := range pkg.Functions {
			fns = append(fns, fnName)
		}
		sort.Strings(fns)
		got[name] = fns
	}
	return got
}

func TestSerializeIsDeterministic(t *testing.T) {
	prgrm := sampleProgramWithPackages(t)
	first, err := cxast.SerializeCXProgram(prgrm, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		again, err := cxast.SerializeCXProgram(prgrm, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, again) {
			diff := 0
			for j := range first {
				if j < len(again) && first[j] != again[j] {
					diff++
				}
			}
			t.Fatalf("serialization %d differs from the first in %d of %d bytes", i+2, diff, len(first))
		}
	}
}

func TestSerializeKeepsFunctionsInTheirPackage(t *testing.T) {
	prgrm := sampleProgramWithPackages(t)
	want := packageFunctions(prgrm)
	for i := 0; i < 10; i++ {
		b, err := cxast.SerializeCXProgram(prgrm, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := cxast.Deserialize(b, nil)
		if err != nil {
			t.Fatal(err)
		}
		if gotFns := packageFunctions(got); !reflect.DeepEqual(gotFns, want) {
			t.Fatalf("round trip %d moved functions between packages:\n got %v\nwant %v", i+1, gotFns, want)
		}
	}
}
