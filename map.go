package goaff4

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/forensicanalysis/goaff4/batch"
)

type affMap struct {
	urn              *url.URL
	dependentStreams map[string]*ImageStream
	entries          []mapEntry
	targets          []string
	size             int64

	next     uint64
	entryPos int64
	bcr      *batch.BufferedChunkReader
}

func (m *affMap) Stat() (fs.FileInfo, error) {
	return m, nil
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
			fmt.Println(target, "all", entry.TargetOffset, entry.Length)

			isChunk := make([]byte, entry.Length)
			// is.Seek(entry.TargetOffset)
			c, err := is.Read(isChunk)
			if err != nil {
				return nil, err
			}
			if uint64(c) != entry.Length {
				panic("wrong length")
			}

			chunk = append(chunk, isChunk...)

			//
			// _, err := is.Read(buf)
			// if err != nil {
			// 	return 0, err
			// }
			// buf = buf[entry.TargetOffset:]
			// c, err = m.buf.Write(buf[:entry.Length])
			// if err != nil {
			// 	return c, err
			// }
		} else {
			return nil, fmt.Errorf("unknown target %s", target)
		}
	}

	m.next = entry.MappedOffset + entry.Length
	m.entryPos = m.entryPos + 1

	return chunk, nil
}

type mapEntry struct {
	MappedOffset uint64
	Length       uint64
	TargetOffset uint64
	TargetID     uint32
}

func newMap(fsys fs.FS, objects map[string]parsedObject, mapURI string) (*affMap, error) {
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

	size, err := strconv.Atoi(objects[mapURI].metadata["size"][0])
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

	return &affMap{
		urn:              mapURL,
		size:             int64(size),
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

func (m *affMap) Name() string {
	return strings.TrimPrefix(m.urn.String(), `aff4://`)
}

func (m *affMap) Size() int64 {
	return m.size
}

func (m *affMap) Mode() fs.FileMode {
	return 0
}

func (m *affMap) ModTime() time.Time {
	return time.Time{}
}

func (m *affMap) IsDir() bool {
	return false
}

func (m *affMap) Sys() interface{} {
	return nil
}

func (m *affMap) Type() fs.FileMode {
	return 0
}

func (m *affMap) Info() (fs.FileInfo, error) {
	return m, nil
}
