package loader

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/skycoin/skycoin-cx/t0-plbolt" // thin: only Add; no depth gain
	"github.com/skycoin/skycoin-cx/t0-textsections"
	"golang.org/x/crypto/blake2b"
)

type File struct {
	FileName   string
	Length     uint32
	Content    []byte
	Blake2Hash string // hex encoded
}

// MarshalBinary returns f in the text form, a [file] section with the length,
// hash and name, and a [content] section with the content as a quoted string.
func (f File) MarshalBinary() ([]byte, error) {
	return textFormEncode([]textsections.TextSection{
		{TextSectionName: "file", TextSectionColumns: []string{"length", "blake2-hash", "name"},
			TextSectionEntries: textFormEntries([]string{strconv.FormatUint(uint64(f.Length), 10), strconv.Quote(f.Blake2Hash), strconv.Quote(f.FileName)})},
		{TextSectionName: "content", TextSectionColumns: []string{"quoted"},
			TextSectionEntries: textFormEntries([]string{strconv.Quote(string(f.Content))})},
	})
}

// UnmarshalBinary reads f from what MarshalBinary wrote.
func (f *File) UnmarshalBinary(data []byte) error {
	sections, err := textFormDecode(data)
	if err != nil {
		return err
	}
	header, err := textFormSection(sections, "file")
	if err != nil {
		return err
	}
	content, err := textFormSection(sections, "content")
	if err != nil {
		return err
	}
	if len(header) != 1 || len(content) != 1 {
		return fmt.Errorf("want one file and one content entry, got %d and %d", len(header), len(content))
	}
	length, err := strconv.ParseUint(header[0][0], 10, 32)
	if err != nil {
		return fmt.Errorf("file length: %w", err)
	}
	var decoded File
	decoded.Length = uint32(length)
	if decoded.Blake2Hash, err = textFormUnquote(header[0][1]); err != nil {
		return err
	}
	if decoded.FileName, err = textFormUnquote(header[0][2]); err != nil {
		return err
	}
	text, err := textFormUnquote(content[0][0])
	if err != nil {
		return err
	}
	decoded.Content = []byte(text)
	*f = decoded
	return nil
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
	return nil
}
