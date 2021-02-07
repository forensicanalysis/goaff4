package goaff4

import (
	"io"
	"io/fs"
	"time"
)

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

	for _, c := range p.volume.objects[p.volume.urn].metadata["contains"] {
		o := p.volume.objects[c]
		if contains(o.metadata["type"], "Image") {
			dsName := o.metadata["dataStream"][0]
			entries = append(entries, &dirEntry{
				uri:      dsName,
				metadata: p.volume.objects[dsName].metadata,
			})
		}
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

	return entries, nil
}

func contains(l []string, s string) bool {
	for _, i := range l {
		if i == s {
			return true
		}
	}
	return false
}

