package goaff4

import (
	"archive/zip"
	"io"
	"io/fs"
)

func New(r io.ReaderAt, s int64) (fs.FS, error) {
	zipr, err := zip.NewReader(r, s)
	if err != nil {
		return nil, err
	}

	volumeURI := readContainerDescription(zipr)
	objects, err := parseObjectsFromRDF(zipr)
	if err != nil {
		return nil, err
	}

	return newVolume(zipr, objects, volumeURI)
}

func readVersionTxt(r fs.FS) string {
	b, _ := fs.ReadFile(r, "version.txt")
	return string(b)
}

func readContainerDescription(r fs.FS) string {
	b, _ := fs.ReadFile(r, "container.description")
	return string(b)
}
