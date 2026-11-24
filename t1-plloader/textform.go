package loader

import (
	"fmt"
	"strconv"
	"strings"

	lz4codec "github.com/skycoin/skycoin-cx/t0-lz4codec"
	"github.com/skycoin/skycoin-cx/t0-textsections"
)

// The database stores a File, Package and PackageList in the CX text form of
// t0-textsections, LZ4 compressed (skycoin-cx#509).

// textFormEncode returns sections as an LZ4 block of their text form.
func textFormEncode(sections []textsections.TextSection) ([]byte, error) {
	text := textsections.TextSectionsWrite(nil, sections)
	var codec lz4codec.Codec
	if err := codec.Init(lz4codec.Config{MaxInput: len(text), MaxOutput: len(text)}); err != nil {
		return nil, err
	}
	defer codec.Close()
	block, err := codec.Compress(text)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), block...), nil
}

// textFormDecode returns the sections of an LZ4 block textFormEncode wrote.
func textFormDecode(block []byte) ([]textsections.TextSection, error) {
	if len(block) == 0 {
		return nil, fmt.Errorf("empty value")
	}
	text, err := textFormText(block)
	if err != nil {
		return nil, err
	}
	return textsections.TextSectionsRead(text)
}

// textFormSection returns the section named name, with each entry split into
// the number of columns of its header. The last column takes the rest.
func textFormSection(sections []textsections.TextSection, name string) ([][]string, error) {
	section, found := textsections.TextSectionFind(sections, name)
	if !found {
		return nil, fmt.Errorf("no [%s] section", name)
	}
	columns := len(section.TextSectionColumns)
	rows := make([][]string, 0, len(section.TextSectionEntries))
	for _, entry := range section.TextSectionEntries {
		row := strings.SplitN(entry.TextSectionEntryText, " ", columns)
		if len(row) != columns {
			return nil, fmt.Errorf("line %d: want %d columns: %q", entry.TextSectionEntryLine, columns, entry.TextSectionEntryText)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// textFormEntries returns entries, each its columns joined by spaces.
func textFormEntries(rows ...[]string) []textsections.TextSectionEntry {
	entries := make([]textsections.TextSectionEntry, len(rows))
	for index, row := range rows {
		entries[index].TextSectionEntryText = strings.Join(row, " ")
	}
	return entries
}

// textFormUnquote returns the Go quoted string text, as strconv.Quote wrote it.
func textFormUnquote(text string) (string, error) {
	value, err := strconv.Unquote(text)
	if err != nil {
		return "", fmt.Errorf("not a quoted string: %q", text)
	}
	return value, nil
}

// textFormText returns the text form in an LZ4 block textFormEncode wrote.
func textFormText(block []byte) ([]byte, error) {
	var codec lz4codec.Codec
	if err := codec.Init(lz4codec.Config{MaxInput: 1, MaxOutput: lz4codec.MaxRatio*len(block) + 64}); err != nil {
		return nil, err
	}
	defer codec.Close()
	text, err := codec.Decompress(block)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), text...), nil
}
