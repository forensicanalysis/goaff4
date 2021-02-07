package goaff4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/forensicanalysis/goaff4/batch"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/golang/snappy"
)

type ImageStream struct {
	fsys        fs.FS
	urn         *url.URL
	chunkSize   int64
	bevyIndexes map[string][]bevyIndexEntry
	size        int64

	// offset int64
	// buf    bytes.Buffer
	bevyNo int64
	bcr      *batch.BufferedChunkReader
}

type bevyIndexEntry struct {
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

	size, err := strconv.Atoi(objects[parseImageURI].metadata["size"][0])
	if err != nil {
		return nil, err
	}

	entries, err := newBevy(fsys, parseImageURI)
	if err != nil {
		return nil, err
	}

	return &ImageStream{
		fsys:        fsys,
		urn:         parseImageURL,
		chunkSize:   int64(chunkSize),
		size:        int64(size),
		bevyIndexes: entries,
	}, nil
}

func (s *ImageStream) Stat() (fs.FileInfo, error) {
	return s, nil
}

func (s *ImageStream) Close() error {
	return nil
}

func (s *ImageStream) Read(b []byte) (int, error) {
	if s.bcr == nil {
		s.bcr = batch.New(s)
	}
	return s.bcr.Read(b)
}

func (s *ImageStream) GetChunk() ([]byte, error) {
	bevyID := strings.Repeat("0", 8-len(fmt.Sprint(s.bevyNo))) + fmt.Sprint(s.bevyNo)

	bevy, err := fs.ReadFile(s.fsys, path.Join(url.QueryEscape(s.urn.String()), bevyID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, io.EOF
		}
		return nil, err
	}
	bevyReader := bytes.NewReader(bevy)

	var chunk []byte

	next := uint64(0)
	for _, bevyIndex := range s.bevyIndexes {
		for _, bevyIndexEntry := range bevyIndex {
			if bevyIndexEntry.BevyOffset != next {
				panic("unorded")
			}
			next += uint64(bevyIndexEntry.ChunkSize)

			_, err := bevyReader.Seek(int64(bevyIndexEntry.BevyOffset), io.SeekStart)
			if err != nil {
				return chunk, err
			}
			compressedChunk := make([]byte, bevyIndexEntry.ChunkSize)
			_, err = bevyReader.Read(compressedChunk)
			if err != nil {
				return chunk, err
			}
			chunk = append(chunk, s.writeChunk(compressedChunk)...)
		}
	}
	s.bevyNo++

	return chunk, nil
}

func (s *ImageStream) writeChunk(compressedChunk []byte) []byte {
	chunk, err := snappy.Decode(nil, compressedChunk)
	if err != nil {
		// TODO remove trial an error decode
		// chunk = compressedChunk
		return compressedChunk
	}

	return chunk
}

func newBevy(fsys fs.FS, parseImageURI string) (map[string][]bevyIndexEntry, error) {
	entries := map[string][]bevyIndexEntry{}
	bevyNo := 0
	for {
		bevyID := strings.Repeat("0", 8-len(fmt.Sprint(bevyNo))) + fmt.Sprint(bevyNo)

		e, err := newBevyIndex(fsys, parseImageURI, bevyID)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			return nil, err
		}
		entries[bevyID] = e
		bevyNo++
	}

	return entries, nil
}

func newBevyIndex(fsys fs.FS, parseImageURI string, name string) ([]bevyIndexEntry, error) {
	bevyIndexFile, err := fsys.Open(path.Join(url.QueryEscape(parseImageURI), name+".index"))
	if err != nil {
		return nil, err
	}
	defer bevyIndexFile.Close()

	var entries []bevyIndexEntry
	for {
		var entry bevyIndexEntry
		err = binary.Read(bevyIndexFile, binary.LittleEndian, &entry)
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

func (s *ImageStream) Name() string {
	return strings.TrimPrefix(s.urn.String(), `aff4://`)
}

func (s *ImageStream) Size() int64 {
	return s.size
}

func (s *ImageStream) Mode() fs.FileMode {
	return 0
}

func (s *ImageStream) ModTime() time.Time {
	return time.Time{}
}

func (s *ImageStream) IsDir() bool {
	return false
}

func (s *ImageStream) Sys() interface{} {
	return nil
}

func (s *ImageStream) Type() fs.FileMode {
	return 0
}

func (s *ImageStream) Info() (fs.FileInfo, error) {
	return s, nil
}
