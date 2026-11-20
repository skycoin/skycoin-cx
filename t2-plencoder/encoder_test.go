package encoder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skycoin/skycoin-cx/t0-plredis"
	"github.com/skycoin/skycoin-cx/t1-plloader"
)

func TestSavePackage(t *testing.T) {
	for _, database := range []string{"redis", "bolt"} {
		t.Run(database, func(t *testing.T) {
			if database == "redis" && redis.Ping() != nil {
				t.Skip("needs a redis server on localhost:6379")
			}
			_, sourceCodes, _ := loader.ParseArgsForCX([]string{"test_tree"}, true)
			if err := loader.LoadCXProgram("Test", sourceCodes, database); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), "out") + "/"
			if err := SavePackagesToDisk("Test", dir, database); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "src", "testimport2", "testimport2file.cx"))
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("test_tree", "src", "testimport2", "testimport2file.cx"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("saved file differs from the source:\n got %q\nwant %q", got, want)
			}
		})
	}
}
