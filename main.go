package main

import (
	"archive/zip"
	"io/fs"
	"log"
	"os"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	r, err := zip.OpenReader("Base-Linear.aff4")
	if err != nil {
		log.Fatal(err)
	}
	if err := printZip(r); err != nil {
		log.Fatal(err)
	}

	readVersionTxt(r)
	volumeURI := readContainerDescription(r)
	objects, err := parseObjectsFromRDF(r)
	if err != nil {
		log.Fatal(err)
	}
	printObjects(objects)

	v, err := newVolume(r, objects, volumeURI)
	if err != nil {
		log.Fatal(err)
	}

	w, err := os.Create("dst")
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()
	_, err = v.WriteTo(w)
	if err != nil {
		log.Fatal(err)
	}
}

func readVersionTxt(r fs.FS) string {
	b, _ := fs.ReadFile(r, "version.txt")
	return string(b)
}

func readContainerDescription(r fs.FS) string {
	b, _ := fs.ReadFile(r, "container.description")
	return string(b)
}
