package graph

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-util"
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func TestGraph(t *testing.T) {
	const want = "id=0, module=main, imports=4,1,2,\nid=1, module=testimport2, imports=2,3,\nid=2, module=testimport1, imports=5,\nid=3, module=testimport3, imports=2,\nid=4, module=os, imports=,\nid=5, module=gl, imports=,\n"
	_, sourceCodes, _, perr := util.ParseArgsForCX([]string{"test_tree"}, true)
	if perr != nil {
		t.Fatal(perr)
	}
	if err := loader.LoadCXProgram("TestTree", sourceCodes); err != nil {
		t.Fatal(err)
	}
	output, err := GetImportGraph("TestTree")
	if err != nil {
		t.Fatal(err)
	}
	if output != want {
		t.Errorf("GetImportGraph:\n got %q\nwant %q", output, want)
	}
}
