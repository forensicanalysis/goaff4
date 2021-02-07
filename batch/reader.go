package batch

import (
	"io"
)

type ChunkReader interface {
	GetChunk() ([]byte, error)
}

type BufferedChunkReader struct {
	cr  ChunkReader
	buf []byte
}

func New(cr ChunkReader) *BufferedChunkReader {
	return &BufferedChunkReader{cr: cr}
}

func (r *BufferedChunkReader) Read(b []byte) (int, error) {
	var chunk []byte
	var err error
	for {
		if len(b) < len(r.buf) {
			n := copy(b, r.buf)
			r.buf = r.buf[len(b):]
			return n, err
		}
		if err != nil {
			n := copy(b, r.buf)
			r.buf = nil
			return n, err
		}

		chunk, err = r.cr.GetChunk()
		if err != nil && err != io.EOF{
			return 0, err
		}
		r.buf = append(r.buf, chunk...)
	}
}
