//go:build !cximageembed

package main

import "os"

// coreImageFileRead reads the image file named by the first argument. A
// build with the cximageembed tag embeds the image instead.
func coreImageFileRead() ([]byte, string) {
	if len(os.Args) < 2 {
		return nil, "give the image file that cx build --target=image wrote"
	}
	file, err := os.ReadFile(os.Args[1])
	if err != nil {
		return nil, err.Error()
	}
	return file, ""
}
