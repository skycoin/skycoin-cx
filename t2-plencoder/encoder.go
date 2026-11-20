package encoder

import (
	"os"

	"github.com/skycoin/skycoin-cx/t0-plbolt"   // thin: only Get; no depth gain
	"github.com/skycoin/skycoin-cx/t0-plredis"  // thin: only Get; no depth gain
	"github.com/skycoin/skycoin-cx/t1-plloader" // thin: only File, Package, PackageList; types only, could move them to t0, tier 2 to 1
)

func SavePackagesToDisk(packageName string, path string, database string) error {
	err := os.Mkdir(path, 0755)
	if err != nil {
		return err
	}

	var packageList loader.PackageList
	var listBytes []byte
	switch database {
	case "redis":
		interfaceString, err := redis.Get(packageName)
		if err != nil {
			return err
		}
		listBytes = []byte(interfaceString.(string))
	case "bolt":
		listBytes, err = bolt.Get(packageName)
		if err != nil {
			return err
		}
	}
	packageList.UnmarshalBinary(listBytes)

	for _, pack := range packageList.Packages {
		var packageStruct loader.Package
		var packageBytes []byte
		switch database {
		case "redis":
			interfaceString, err := redis.Get(pack)
			if err != nil {
				return err
			}
			packageBytes = []byte(interfaceString.(string))
		case "bolt":
			packageBytes, err = bolt.Get(pack)
			if err != nil {
				return err
			}
		}
		err := packageStruct.UnmarshalBinary(packageBytes)
		if err != nil {
			return err
		}

		var filePath string
		if packageStruct.PackageName == "main" {
			filePath = path + "src/"
		} else {
			filePath = path + "src/" + packageStruct.PackageName + "/"
		}

		err = SaveFilesToDisk(packageStruct, filePath, database)
		if err != nil {
			return err
		}
	}
	return nil
}

func SaveFilesToDisk(packageStruct loader.Package, filePath string, database string) error {
	err := os.MkdirAll(filePath, 0755)
	if err != nil {
		return err
	}
	for _, file := range packageStruct.Files {
		var fileStruct loader.File
		var fileBytes []byte
		switch database {
		case "redis":
			interfaceString, err := redis.Get(file)
			if err != nil {
				return err
			}
			fileBytes = []byte(interfaceString.(string))
		case "bolt":
			fileBytes, err = bolt.Get(file)
			if err != nil {
				return err
			}
		}
		fileStruct.UnmarshalBinary(fileBytes)

		err = os.WriteFile(filePath+fileStruct.FileName, fileStruct.Content, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
