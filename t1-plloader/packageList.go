package loader

import (
	"errors"
	"fmt"
	"github.com/skycoin/skycoin-cx/t0-textsections"
)

type PackageList struct {
	Packages []string
}

// MarshalBinary returns pl in the text form, a [packages] section with the
// key of each package.
func (pl PackageList) MarshalBinary() ([]byte, error) {
	packages := make([][]string, len(pl.Packages))
	for index, key := range pl.Packages {
		packages[index] = []string{key}
	}
	return textFormEncode([]textsections.TextSection{
		{TextSectionName: "packages", TextSectionColumns: []string{"key"}, TextSectionEntries: textFormEntries(packages...)},
	})
}

// UnmarshalBinary reads pl from what MarshalBinary wrote.
func (pl *PackageList) UnmarshalBinary(data []byte) error {
	sections, err := textFormDecode(data)
	if err != nil {
		return err
	}
	packages, err := textFormSection(sections, "packages")
	if err != nil {
		return err
	}
	var decoded PackageList
	for _, row := range packages {
		decoded.Packages = append(decoded.Packages, row[0])
	}
	*pl = decoded
	return nil
}

// Encode a package and put it in the specified package list
func (pl *PackageList) appendPackage(newPackage *Package) error {
	if newPackage == nil {
		return errors.New("append package: nil package")
	}
	hash, err := newPackage.getHash()
	if err != nil {
		return err
	}
	if Contains(pl.Packages, fmt.Sprintf("%x", hash[:])) {
		return nil
	}
	pl.Packages = append(pl.Packages, fmt.Sprintf("%x", hash[:]))
	err = newPackage.saveToDatabase(hash)
	if err != nil {
		return err
	}
	return nil
}
