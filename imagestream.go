package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/golang/snappy"
)

type ImageStream struct {
	fsys      fs.FS
	urn       *url.URL
	chunkSize int64
	entries   map[string][]indexEntry
}

type indexEntry struct {
	BevyOffset uint64
	ChunkSize  uint32
}

func newImageStream(fsys fs.FS, objects map[string]parsedObject, parseImageURI string) (*ImageStream, error) {
	parseImageURL, err := url.Parse(parseImageURI)
	if err != nil {
		return nil, err
	}
	chunkSize, err := strconv.Atoi(objects[parseImageURI].metadata["chunkSize"][0])
	if err != nil {
		return nil, err
	}

	entries, err := newIndexEntries(fsys, parseImageURI)
	if err != nil {
		return nil, err
	}

	return &ImageStream{
		fsys:      fsys,
		urn:       parseImageURL,
		chunkSize: int64(chunkSize),
		entries:   entries,
	}, nil
}

func (s ImageStream) WriteTo(w io.Writer) (int64, error) {
	chunkId := 0
	offset := 0
	for {
		name := strings.Repeat("0", 8-len(fmt.Sprint(chunkId))) + fmt.Sprint(chunkId)

		bevyBytes, err := fs.ReadFile(s.fsys, path.Join(url.QueryEscape(s.urn.String()), name))
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return 0, err
		}
		bevyReader := bytes.NewReader(bevyBytes)

		for _, entries := range s.entries {
			for _, entry := range entries {
				_, err := bevyReader.Seek(int64(entry.BevyOffset), io.SeekStart)
				if err != nil {
					return 0, err
				}
				b := make([]byte, entry.ChunkSize)
				read, err := bevyReader.Read(b)
				if err != nil {
					return 0, err
				}
				var dec []byte
				if int(entry.ChunkSize) == read {
					dec = b
				} else {
					dec, err = snappy.Decode(nil, b)
					if err != nil {
						return 0, err
					}
					fmt.Println("decode done")
				}

				n, err := w.Write(dec)
				if err != nil {
					return 0, err
				}
				offset += n
			}
		}
		chunkId += 1
	}
	return int64(offset), nil
}

func newIndexEntries(fsys fs.FS, parseImageURI string) (map[string][]indexEntry, error) {
	entries := map[string][]indexEntry{}
	chunkId := 0
	for {
		name := strings.Repeat("0", 8-len(fmt.Sprint(chunkId))) + fmt.Sprint(chunkId)

		e, err := newBrevy(fsys, parseImageURI, name)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return nil, err
		}
		entries[name] = e
		chunkId += 1
	}

	return entries, nil
}

func newBrevy(fsys fs.FS, parseImageURI string, name string) ([]indexEntry, error) {
	f, err := fsys.Open(path.Join(url.QueryEscape(parseImageURI), name+".index"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []indexEntry
	for {
		var entry indexEntry
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
