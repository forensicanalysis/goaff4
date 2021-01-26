package main

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"strings"
)

func printZip(r *zip.ReadCloser) error {
	return fs.WalkDir(r, ".", func(path string, d fs.DirEntry, err error) error {
		ii, _ := d.Info()
		fmt.Println(path, ii.Size())
		return nil
	})
}

func printObjects(objects map[string]parsedObject) {
	for _, o := range objects {
		fmt.Printf("<%s>\n", o.urn)
		for k, v := range o.metadata {
			fmt.Printf("\t%-30s %s\n", k, "'"+strings.Join(v, ", ")+"'")
		}
		fmt.Println()
	}
}
