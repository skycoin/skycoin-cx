package bolt

import (
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// VIOLATION: state outside State. Owner: t0-plbolt; moves to PackageDBMng in phase 2.
var db *bolt.DB

// Init is init 1 of the package loader tools. It opens the database file at
// path, creating it and the program bucket if needed, and keeps it open until
// Close. There is no default path, the caller chooses where the file lives.
func Init(path string) error {
	if path == "" {
		return errors.New("bolt: empty database path")
	}
	if db != nil {
		return fmt.Errorf("bolt: %s is already open, call Close first", db.Path())
	}
	opened, err := bolt.Open(path, 0644, nil)
	if err != nil {
		return fmt.Errorf("bolt: open %s: %w", path, err)
	}
	err = opened.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("program"))
		return err
	})
	if err != nil {
		opened.Close()
		return fmt.Errorf("bolt: create bucket in %s: %w", path, err)
	}
	db = opened
	return nil
}

// Close closes the database opened by Init.
func Close() error {
	if db == nil {
		return errors.New("bolt: database is not open")
	}
	err := db.Close()
	db = nil
	return err
}

func open() (*bolt.DB, error) {
	if db == nil {
		return nil, errors.New("bolt: database is not open, call Init first")
	}
	return db, nil
}

// Add stores value under key in the program bucket.
func Add(key string, value []byte) error {
	if key == "" {
		return errors.New("bolt: put: empty key")
	}
	db, err := open()
	if err != nil {
		return err
	}
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("program"))
		if bucket == nil {
			return errors.New("bolt: bucket program was not found")
		}
		if err := bucket.Put([]byte(key), value); err != nil {
			return fmt.Errorf("bolt: put %q: %w", key, err)
		}
		return nil
	})
}

// Get returns a copy of the value stored under key. A missing key is an error.
func Get(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("bolt: get: empty key")
	}
	db, err := open()
	if err != nil {
		return nil, err
	}
	var ret []byte
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("program"))
		if bucket == nil {
			return errors.New("bolt: bucket program was not found")
		}
		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("bolt: no value for key %q", key)
		}
		ret = make([]byte, len(value))
		copy(ret, value)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
