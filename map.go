package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

type Map struct {
	urn              *url.URL
	dependentStreams map[string]*ImageStream
	entries          []mapEntry
	targets          []string
}

func (m *Map) WriteTo(w io.Writer) (n int64, err error) {
	next := uint64(0)
	for _, entry := range m.entries {
		if entry.MappedOffset != next {
			panic("unorded")
		}

		c := 0
		target := m.targets[entry.TargetID]
		switch {
		case target == "http://aff4.org/Schema#Zero":
			c, err = w.Write(bytes.Repeat([]byte{0x00}, int(entry.Length)))
			if err != nil {
				return 0, err
			}
		case strings.HasPrefix(target, "http://aff4.org/Schema#SymbolicStream61"):
			c, err = w.Write(bytes.Repeat([]byte{0x61}, int(entry.Length)))
			if err != nil {
				return 0, err
			}
		case strings.HasPrefix(target, "http://aff4.org/Schema#SymbolicStreamFF"):
			c, err = w.Write(bytes.Repeat([]byte{0xFF}, int(entry.Length)))
			if err != nil {
				return 0, err
			}
		default:
			if is, ok := m.dependentStreams[target]; ok {
				buf := &bytes.Buffer{}
				_, err = is.WriteTo(buf)
				if err != nil {
					return 0, err
				}
				x := buf.Next(int(entry.TargetOffset))
				if uint64(len(x)) != entry.TargetOffset {
					// log.Fatal("insuff skip ", c, entry.TargetOffset)
				}
				c, err = w.Write(buf.Next(int(entry.Length)))
				if err != nil {
					return 0, err
				}
				if uint64(c) != entry.Length {
					// log.Fatal("insuff read ", c, entry.Length)
				}
			} else {
				return 0, fmt.Errorf("unknown target %s", target)
			}
		}

		next = entry.MappedOffset + entry.Length
		n += int64(c)
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

	imageStreams := map[string]*ImageStream{}
	for _, dependentStreamURI := range objects[mapURI].metadata["dependentStream"] {
		imageStream, err := newImageStream(fsys, objects, dependentStreamURI)
		if err != nil {
			return nil, err
		}

		imageStreams[dependentStreamURI] = imageStream
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
