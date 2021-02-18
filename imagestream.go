package goaff4

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/forensicanalysis/goaff4/batch"
	"github.com/golang/snappy"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"strings"
)

type imageStream struct {
	info *dirEntry

	fsys fs.FS

	bevyIndexes map[string][]bevyIndexEntry

	bevyNo int64
	bcr    *batch.BufferedChunkReader
}

type bevyIndexEntry struct {
	BevyOffset uint64
	ChunkSize  uint32
}

func newImageStream(zipfs *zip.Reader, objects map[string]parsedObject, parseImageURI string) (*imageStream, error) {
	entries, err := newBevy(zipfs, parseImageURI)
	if err != nil {
		return nil, err
	}

	return &imageStream{
		info: &dirEntry{
			uri:      parseImageURI,
			metadata: objects[parseImageURI].metadata,
		},
		fsys:        zipfs,
		bevyIndexes: entries,
	}, nil
}

func (s *imageStream) Stat() (fs.FileInfo, error) {
	return s.info, nil
}

func (s *imageStream) Close() error {
	return nil
}

func (s *imageStream) Read(b []byte) (int, error) {
	if s.bcr == nil {
		s.bcr = batch.New(s)
	}
	return s.bcr.Read(b)
}

func (s *imageStream) GetChunk() ([]byte, error) {
	bevyID := strings.Repeat("0", 8-len(fmt.Sprint(s.bevyNo))) + fmt.Sprint(s.bevyNo)

	bevy, err := fs.ReadFile(s.fsys, path.Join(url.QueryEscape(s.info.uri), bevyID))
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

func (s *imageStream) writeChunk(compressedChunk []byte) []byte {
	chunk, err := snappy.Decode(nil, compressedChunk)
	if err != nil {
		// TODO remove trial an error decode
		// chunk = compressedChunk
		return compressedChunk
	}

	return chunk
}

func newBevy(zipfs *zip.Reader, parseImageURI string) (map[string][]bevyIndexEntry, error) {
	entries := map[string][]bevyIndexEntry{}
	bevyNo := 0
	for {
		bevyID := strings.Repeat("0", 8-len(fmt.Sprint(bevyNo))) + fmt.Sprint(bevyNo)

		e, err := newBevyIndex(zipfs, parseImageURI, bevyID)
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

func newBevyIndex(zipfs *zip.Reader, parseImageURI string, name string) ([]bevyIndexEntry, error) {
	bevyIndexFile, err := zipfs.Open(path.Join(url.QueryEscape(parseImageURI), name+".index"))
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
