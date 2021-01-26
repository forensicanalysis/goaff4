package main

import (
	"io"
	"io/fs"
	"net/url"
)

type Volume struct {
	urn         *url.URL
	containsMap []*Map
}

func newVolume(fsys fs.FS, objects map[string]parsedObject, volumeURI string) (*Volume, error) {
	volumeURL, err := url.Parse(volumeURI)
	if err != nil {
		return nil, err
	}
	vo := objects[volumeURI]
	v := &Volume{
		urn: volumeURL,
	}
	for _, c := range vo.metadata["contains"] {
		o := objects[c]
		switch o.metadata["type"][0] {
		case "Map":
			m, err := newMap(fsys, objects, c)
			if err != nil {
				return nil, err
			}
			v.containsMap = append(v.containsMap, m)
		}
	}
	return v, nil
}

func (v *Volume) WriteTo(w io.Writer) (n int64, err error) {
	for _, m := range v.containsMap {
		c, err := m.WriteTo(w)
		if err != nil {
			return 0, err
		}
		n += c
	}
	return n, nil
}
