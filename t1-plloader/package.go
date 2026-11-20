package loader

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Add; no depth gain
	"golang.org/x/crypto/blake2b"
)

type Package struct {
	PackageName string
	Blake2Hash  string // hex encoded
	Files       []string
}

func (p Package) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Package) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

// Encode a file and put it in the specified package
func (p *Package) appendFile(newFile *File) error {
	if newFile == nil {
		return errors.New("append file: nil file")
	}
	hash, err := newFile.getHash()
	if err != nil {
		return err
	}
	p.Files = append(p.Files, fmt.Sprintf("%x", hash[:]))
	err = newFile.saveToDatabase(hash)
	if err != nil {
		return err
	}
	return nil
}

func (p *Package) getHash() ([64]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(p)
	if err != nil {
		return [64]byte{}, err
	}
	return blake2b.Sum512(buffer.Bytes()), err
}

func (p *Package) saveToDatabase(hash [64]byte) error {
	key := fmt.Sprintf("%x", hash[:])
	value, err := p.MarshalBinary()
	if err != nil {
		return fmt.Errorf("encode package %s: %w", p.PackageName, err)
	}
	if err := bolt.Add(key, value); err != nil {
		return fmt.Errorf("save package %s: %w", p.PackageName, err)
	}
	PackageHashMap[p.PackageName] = key
	return nil
}
