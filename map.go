package goaff4

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type affMap struct {
	urn              *url.URL
	dependentStreams map[string]*ImageStream
	entries          []mapEntry
	targets          []string
	size             int64
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func (m *affMap) Read(p []byte) (int, error) {
	b := &bytes.Buffer{}
	_, err := m.WriteTo(b)
	if err != nil {
		return 0, err
	}

	x := max(len(p), b.Len())
	return copy(p, b.Bytes()[:x]), nil
}

func (m *affMap) Stat() (fs.FileInfo, error) {
	return m, nil
}

func (m *affMap) Close() error {
	return nil
}

func (m *affMap) WriteTo(w io.Writer) (n int64, err error) {
	next := uint64(0)
	for _, entry := range m.entries {
		if entry.MappedOffset != next {
			return 0, errors.New("unordered list")
		}

		c := 0
		target := m.targets[entry.TargetID]
		switch {
		case target == "http://aff4.org/Schema#UnknownData":
			fallthrough
		case target == "http://aff4.org/Schema#Zero":
			c, err = w.Write(bytes.Repeat([]byte{0x00}, int(entry.Length)))
			if err != nil {
				return 0, err
			}
		case strings.HasPrefix(target, "http://aff4.org/Schema#SymbolicStream"):
			s := strings.TrimPrefix(target, "http://aff4.org/Schema#SymbolicStream")
			decoded, err := hex.DecodeString(s)
			if err != nil {
				return 0, err
			}
			c, err = w.Write(bytes.Repeat(decoded[:1], int(entry.Length)))
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
				buf.Next(int(entry.TargetOffset))
				c, err = w.Write(buf.Next(int(entry.Length)))
				if err != nil {
					return 0, err
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
