package goaff4

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"strings"
	"time"
)

type superFile interface {
	fs.File
	fs.DirEntry
}

type volume struct {
	urn     *url.URL
	items   map[string]superFile
	objects map[string]parsedObject
	fsys    fs.FS
}

func newVolume(fsys fs.FS, objects map[string]parsedObject, volumeURI string) (*volume, error) {
	volumeURL, err := url.Parse(volumeURI)
	if err != nil {
		return nil, err
	}
	vo := objects[volumeURI]
	v := &volume{
		fsys:    fsys,
		urn:     volumeURL,
		items:   map[string]superFile{},
		objects: objects,
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
	fmt.Println("create", c)
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
	fmt.Println("open", name)
	if name == "." {
		return &pseudoRoot{volume: v}, nil
	}
	name = strings.Replace(name, "aff4%3A%2F%2F", "", 1)
	if _, ok := v.objects["aff4://"+name]; ok {
		return createItem(v.fsys, v.objects, "aff4://"+name)
		// return f, nil
	}
	fmt.Println(v.objects)
	return nil, fmt.Errorf("%w: %s", fs.ErrNotExist, name)
}

type pseudoRoot struct {
	readDirPos int
	volume     *volume
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
	for _, item := range p.volume.items {
		entries = append(entries, item)
	}

	if p.readDirPos >= len(entries) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}

	if n > 0 && p.readDirPos+n <= len(entries) {
		entries = entries[p.readDirPos : p.readDirPos+n]
		p.readDirPos += n
	} else {
		entries = entries[p.readDirPos:]
		p.readDirPos += len(entries)
	}

	fmt.Println("readdir", entries)
	return entries, nil
}
