package main

import (
	"encoding/binary"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

type Map struct {
	urn              *url.URL
	dependentStreams []*ImageStream
	entries          []mapEntry
	targets          []string
}

func (m *Map) WriteTo(w io.Writer) (n int64, err error) {
	// fmt.Println(entry.MappedOffset, entry.Length, entry.TargetID, entry.TargetOffset) TODO

	for _, s := range m.dependentStreams {
		c, err := s.WriteTo(w)
		if err != nil {
			return 0, err
		}
		n += c
	}
	return n, nil
}

type mapEntry struct {
	MappedOffset uint64
	Length       uint64
	TargetOffset uint64
	TargetID     uint32
}

func newMap(fsys fs.FS, objects map[string]parsedObject, mapURI string) (*Map, error) {
	mapURL, err := url.Parse(mapURI)
	if err != nil {
		return nil, err
	}

	targets, err := newTargetEntries(fsys, mapURI)
	if err != nil {
		return nil, err
	}

	entries, err := newMapEntries(fsys, mapURI)
	if err != nil {
		return nil, err
	}

	var imageStreams []*ImageStream
	for _, dependentStreamURI := range objects[mapURI].metadata["dependentStream"] {
		imageStream, err := newImageStream(fsys, objects, dependentStreamURI)
		if err != nil {
			return nil, err
		}

		imageStreams = append(imageStreams, imageStream)
	}
	return &Map{
		urn:              mapURL,
		entries:          entries,
		targets:          targets,
		dependentStreams: imageStreams,
	}, nil
}

func newTargetEntries(fsys fs.FS, mapURI string) ([]string, error) {
	f, err := fsys.Open(path.Join(url.QueryEscape(mapURI), "idx"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(b), "\n"), nil
}

func newMapEntries(fsys fs.FS, mapURI string) ([]mapEntry, error) {
	f, err := fsys.Open(path.Join(url.QueryEscape(mapURI), "map"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []mapEntry
	for {
		var entry mapEntry
		err = binary.Read(f, binary.LittleEndian, &entry)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
