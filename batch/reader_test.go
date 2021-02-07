package batch

import (
	"io"
	"testing"
	"testing/iotest"
)

type fiveByteReader struct {
	total int
	count int
}

func (f *fiveByteReader) GetChunk() ([]byte, error) {
	if f.count >= f.total {
		return nil, io.EOF
	}
	f.count++
	return []byte("abcde"), nil
}

func TestBufferedChunkReader_Read(t *testing.T) {
	type fields struct {
		cr ChunkReader
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{"simple", fields{cr: &fiveByteReader{total: 2}}, []byte("abcdeabcde")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BufferedChunkReader{cr: tt.fields.cr}

			err := iotest.TestReader(r, tt.want)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
