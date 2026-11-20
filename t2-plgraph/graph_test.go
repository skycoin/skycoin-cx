package graph

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plredis"
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func TestGraph(t *testing.T) {
	const want = "id=0, module=main, imports=4,2,1,\nid=1, module=testimport1, imports=5,\nid=2, module=testimport2, imports=1,3,\nid=3, module=testimport3, imports=1,\nid=4, module=os, imports=,\nid=5, module=gl, imports=,\n"
	for _, database := range []string{"redis", "bolt"} {
		t.Run(database, func(t *testing.T) {
			if database == "redis" && redis.Ping() != nil {
				t.Skip("needs a redis server on localhost:6379")
			}
			_, sourceCodes, _ := loader.ParseArgsForCX([]string{"test_tree"}, true)
			if err := loader.LoadCXProgram("TestTree", sourceCodes, database); err != nil {
				t.Fatal(err)
			}
			output, err := GetImportGraph("TestTree", database)
			if err != nil {
				t.Fatal(err)
			}
			if output != want {
				t.Errorf("GetImportGraph:\n got %q\nwant %q", output, want)
			}
		})
	}
}
