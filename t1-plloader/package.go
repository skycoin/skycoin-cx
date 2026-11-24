package loader

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Add; no depth gain
	"github.com/skycoin/skycoin-cx/t0-textsections"
	"golang.org/x/crypto/blake2b"
)

type Package struct {
	PackageName string
	Blake2Hash  string // hex encoded
	Files       []string
}

// MarshalBinary returns p in the text form, a [package] section with the hash
// and name, and a [files] section with the key of each file.
func (p Package) MarshalBinary() ([]byte, error) {
	files := make([][]string, len(p.Files))
	for index, key := range p.Files {
		files[index] = []string{key}
	}
	return textFormEncode([]textsections.TextSection{
		{TextSectionName: "package", TextSectionColumns: []string{"blake2-hash", "name"},
			TextSectionEntries: textFormEntries([]string{strconv.Quote(p.Blake2Hash), strconv.Quote(p.PackageName)})},
		{TextSectionName: "files", TextSectionColumns: []string{"key"}, TextSectionEntries: textFormEntries(files...)},
	})
}

// UnmarshalBinary reads p from what MarshalBinary wrote.
func (p *Package) UnmarshalBinary(data []byte) error {
	sections, err := textFormDecode(data)
	if err != nil {
		return err
	}
	header, err := textFormSection(sections, "package")
	if err != nil {
		return err
	}
	files, err := textFormSection(sections, "files")
	if err != nil {
		return err
	}
	if len(header) != 1 {
		return fmt.Errorf("want one package entry, got %d", len(header))
	}
	var decoded Package
	if decoded.Blake2Hash, err = textFormUnquote(header[0][0]); err != nil {
		return err
	}
	if decoded.PackageName, err = textFormUnquote(header[0][1]); err != nil {
		return err
	}
	for _, row := range files {
		decoded.Files = append(decoded.Files, row[0])
	}
	*p = decoded
	return nil
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
	return nil
}
