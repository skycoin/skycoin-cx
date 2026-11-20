package loader

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Add; no depth gain
	"golang.org/x/crypto/blake2b"
)

type File struct {
	FileName   string
	Length     uint32
	Content    []byte
	Blake2Hash string // hex encoded
}

func (f File) MarshalBinary() ([]byte, error) {
	return json.Marshal(f)
}

func (f *File) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, f)
}

func (f *File) getHash() ([64]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(f)
	if err != nil {
		return [64]byte{}, err
	}
	return blake2b.Sum512(buffer.Bytes()), err
}

func (f *File) saveToDatabase(hash [64]byte) error {
	key := fmt.Sprintf("%x", hash[:])
	value, err := f.MarshalBinary()
	if err != nil {
		return fmt.Errorf("encode file %s: %w", f.FileName, err)
	}
	if err := bolt.Add(key, value); err != nil {
		return fmt.Errorf("save file %s: %w", f.FileName, err)
	}
	FileHashMap[f.FileName] = key
	return nil
}
