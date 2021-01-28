package goaff4

import (
	"fmt"
	"io/fs"
	"net/url"
	"time"
)

type superFile interface {
	fs.File
	fs.DirEntry
}

type volume struct {
	urn   *url.URL
	items map[string]superFile
}

func newVolume(fsys fs.FS, objects map[string]parsedObject, volumeURI string) (*volume, error) {
	volumeURL, err := url.Parse(volumeURI)
	if err != nil {
		return nil, err
	}
	vo := objects[volumeURI]
	v := &volume{
		urn:   volumeURL,
		items: map[string]superFile{},
	}
	for _, c := range vo.metadata["contains"] {
		o := objects[c]
		if contains(o.metadata["type"], "Image") {
			item, err := createItem(fsys, objects, o.metadata["dataStream"][0])
			if err != nil {
				return nil, err
			}
			v.items[item.Name()] = item
		}
	}
	return v, nil
}

func contains(l []string, s string) bool {
	for _, i := range l {
		if i == s {
			return true
		}
	}
	return false
}

func createItem(fsys fs.FS, objects map[string]parsedObject, c string) (superFile, error) {
	o := objects[c]
	switch o.metadata["type"][0] {
	case "Map":
		return newMap(fsys, objects, c)
	case "ImageStream":
		return newImageStream(fsys, objects, c)
	}
	return nil, fmt.Errorf("unknown type %s", o.metadata["type"])
}

func (v *volume) Open(name string) (fs.File, error) {
	if name == "." {
		return &pseudoRoot{v}, nil
	}
	fmt.Println(v.items)
	if f, ok := v.items[name]; ok {
		return f, nil
	}
	return nil, fs.ErrNotExist
}

type pseudoRoot struct {
	volume *volume
}

func (p *pseudoRoot) Name() string { return "." }

func (p *pseudoRoot) Size() int64 { return 0 }

func (p *pseudoRoot) Mode() fs.FileMode { return fs.ModeDir }

func (p *pseudoRoot) ModTime() time.Time { return time.Time{} }

func (p *pseudoRoot) IsDir() bool { return true }

func (p *pseudoRoot) Sys() interface{} { return nil }

func (p *pseudoRoot) Stat() (fs.FileInfo, error) { return p, nil }

func (p *pseudoRoot) Read([]byte) (int, error) { return 0, fs.ErrInvalid }

func (p *pseudoRoot) Close() error { return nil }

func (p *pseudoRoot) ReadDir(n int) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	i := 0
	for _, item := range p.volume.items {
		if n != -1 && i > n {
			break
		}
		entries = append(entries, item)
		i += 1
	}
	return entries, nil
}
