package bolt

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	bbolt "go.etcd.io/bbolt"
)

func TestAddGet(t *testing.T) {
	if err := Add("bolt-test-key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	got, err := Get("bolt-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "value" {
		t.Errorf("Get: got %q, want %q", got, "value")
	}
}

func TestGetMissingKey(t *testing.T) {
	if _, err := Get("bolt-test-missing-key"); err == nil {
		t.Error("Get of a missing key: want an error, got nil")
	}
}

func TestEmptyKey(t *testing.T) {
	if err := Add("", []byte("v")); err == nil {
		t.Error("Add with an empty key: want an error, got nil")
	}
	if _, err := Get(""); err == nil {
		t.Error("Get with an empty key: want an error, got nil")
	}
}

func TestDatabaseAtGivenPathHeldOpen(t *testing.T) {
	if err := Add("k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(exe), "program_list.db")); err == nil {
		t.Errorf("program_list.db was written next to the executable in %s", filepath.Dir(exe))
	}
	if _, err := os.Stat(testDBPath); err != nil {
		t.Errorf("no database at the path given to Init: %v", err)
	}
	other, err := bbolt.Open(testDBPath, 0644, &bbolt.Options{Timeout: 100 * time.Millisecond})
	if err == nil {
		other.Close()
		t.Error("the database is not held open between calls, a second open got the lock")
	}
}

func TestInitNeedsPath(t *testing.T) {
	if err := Init(""); err == nil {
		t.Error("Init with an empty path: want an error, got nil")
	}
	if err := Init(testDBPath); err == nil {
		t.Error("Init while open: want an error, got nil")
	}
}
