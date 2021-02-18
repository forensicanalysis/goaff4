package goaff4

import (
	"archive/zip"
	"fmt"
	"io/fs"
	"net/url"
)

type volume struct {
	urn string

	objects map[string]parsedObject
	zipfs   *zip.Reader
}

func newVolume(zipfs *zip.Reader, objects map[string]parsedObject, volumeURI string) (*volume, error) {
	_, err := url.Parse(volumeURI)
	if err != nil {
		return nil, err
	}

	return &volume{
		zipfs:   zipfs,
		urn:     volumeURI,
		objects: objects,
	}, nil
}

func (v *volume) Open(name string) (fs.File, error) {
	if name == "." {
		return &pseudoRoot{volume: v}, nil
	}
	name = "aff4://" + name
	if o, ok := v.objects[name]; ok {
		switch o.metadata["type"][0] {
		case "Map":
			return newMap(v.zipfs, v.objects, name)
		case "imageStream":
			return newImageStream(v.zipfs, v.objects, name)
		}
		return nil, fmt.Errorf("unknown type %s", o.metadata["type"])
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
}

func (v *volume) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		return &pseudoRoot{volume: v}, nil
	}
	name = "aff4://" + name
	if o, ok := v.objects[name]; ok {
		return &dirEntry{uri: name, metadata: o.metadata}, nil
	}
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
}
