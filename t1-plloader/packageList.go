package loader

import (
	"encoding/json"
	"errors"
	"fmt"
)

type PackageList struct {
	Packages []string
}

func (pl PackageList) MarshalBinary() ([]byte, error) {
	return json.Marshal(pl)
}

func (pl *PackageList) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, pl)
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
