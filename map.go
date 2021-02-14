package goaff4

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/forensicanalysis/goaff4/batch"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

type affMap struct {
	info *dirEntry

	dependentStreams map[string]*ImageStream
	entries          []mapEntry

	targets  []string
	next     uint64
	entryPos int64
	bcr      *batch.BufferedChunkReader
}

type mapEntry struct {
	MappedOffset uint64
	Length       uint64
	TargetOffset uint64
	TargetID     uint32
}

func (m *affMap) Stat() (fs.FileInfo, error) {
	return m.info, nil
}

func (m *affMap) Close() error {
	return nil
}

func (m *affMap) Read(b []byte) (int, error) {
	if m.bcr == nil {
		m.bcr = batch.New(m)
	}
	return m.bcr.Read(b)
}

func (m *affMap) GetChunk() ([]byte, error) {
	if m.entryPos > int64(len(m.entries)) {
		return nil, io.EOF
	}

	entry := m.entries[m.entryPos]

	if entry.MappedOffset != m.next {
		return nil, fmt.Errorf("unordered list: %d %d", entry.MappedOffset, m.next)
	}

	var chunk []byte
	target := m.targets[entry.TargetID]
	switch {
	case target == "http://aff4.org/Schema#UnknownData":
		fallthrough
	case target == "http://aff4.org/Schema#Zero":
		chunk = bytes.Repeat([]byte{0x00}, int(entry.Length))
	case strings.HasPrefix(target, "http://aff4.org/Schema#SymbolicStream"):
		s := strings.TrimPrefix(target, "http://aff4.org/Schema#SymbolicStream")
		decoded, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}
		chunk = bytes.Repeat(decoded[:1], int(entry.Length))
	default:
		if is, ok := m.dependentStreams[target]; ok {
			isChunk := make([]byte, entry.Length)
			c, err := is.Read(isChunk)
			if err != nil {
				return nil, err
			}
			if uint64(c) != entry.Length {
				return nil, errors.New("wrong length")
			}

			chunk = append(chunk, isChunk...)
		} else {
			return nil, fmt.Errorf("unknown target %s", target)
		}
	}

	m.next = entry.MappedOffset + entry.Length
	m.entryPos++

	return chunk, nil
}

func newMap(zipfs *zip.Reader, objects map[string]parsedObject, mapURI string) (*affMap, error) {
	targets, err := newTargetEntries(zipfs, mapURI)
	if err != nil {
		return nil, err
	}

	entries, err := newMapEntries(zipfs, mapURI)
	if err != nil {
		return nil, err
	}

	imageStreams := map[string]*ImageStream{}
	for _, dependentStreamURI := range objects[mapURI].metadata["dependentStream"] {
		imageStream, err := newImageStream(zipfs, objects, dependentStreamURI)
		if err != nil {
			return nil, err
		}

		imageStreams[dependentStreamURI] = imageStream
	}

	return &affMap{
		info:             &dirEntry{uri: mapURI, metadata: objects[mapURI].metadata},
		entries:          entries,
		targets:          targets,
		dependentStreams: imageStreams,
	}, nil
}

func newTargetEntries(zipfs *zip.Reader, mapURI string) ([]string, error) {
	f, err := zipfs.Open(path.Join(url.QueryEscape(mapURI), "idx"))
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

func newMapEntries(zipfs *zip.Reader, mapURI string) ([]mapEntry, error) {
	f, err := zipfs.Open(path.Join(url.QueryEscape(mapURI), "map"))
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
