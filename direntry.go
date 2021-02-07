package goaff4

import (
	"io/fs"
	"strconv"
	"strings"
	"time"
)

type dirEntry struct {
	uri      string
	metadata map[string][]string
}

func (m *dirEntry) Name() string {
	return strings.TrimPrefix(m.uri, `aff4://`)
}

func (m *dirEntry) Size() int64 {
	size, err := strconv.Atoi(m.metadata["size"][0])
	if err != nil {
		return 0
	}
	return int64(size)
}

func (m *dirEntry) Mode() fs.FileMode {
	return 0
}

func (m *dirEntry) ModTime() time.Time {
	return time.Time{}
}

func (m *dirEntry) IsDir() bool {
	return false
}

func (m *dirEntry) Sys() interface{} {
	return nil
}

func (m *dirEntry) Type() fs.FileMode {
	return 0
}

func (m *dirEntry) Info() (fs.FileInfo, error) {
	return m, nil
}

