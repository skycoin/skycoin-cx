package tree

import (
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plredis"
	"github.com/skycoin/skycoin-cx/t1-plloader2"
)

func TestTree(t *testing.T) {
	const want = "main\n|--os\n|--testimport2\n|  |--testimport1\n|  |  `--gl\n|  `--testimport3\n|     `--testimport1\n|        `--gl\n`--testimport1\n   `--gl\n"
	for _, database := range []string{"redis", "bolt"} {
		t.Run(database, func(t *testing.T) {
			if database == "redis" && redis.Ping() != nil {
				t.Skip("needs a redis server on localhost:6379")
			}
			_, sourceCodes, _ := loader.ParseArgsForCX([]string{"test_tree"}, true)
			if err := loader.LoadCXProgram("TestTree", sourceCodes, database); err != nil {
				t.Fatal(err)
			}
			output, err := GetImportTree("TestTree", database)
			if err != nil {
				t.Fatal(err)
			}
			if output != want {
				t.Errorf("GetImportTree:\n got %q\nwant %q", output, want)
			}
		})
	}
}
