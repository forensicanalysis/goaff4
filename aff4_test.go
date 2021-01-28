package goaff4

import (
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"testing/fstest"
)

func loadFile(p string) (io.ReaderAt, int64, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}
	return f, info.Size(), nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		wantErr bool
	}{
		{"Base-Linear.aff4", []string{"fcbfdce7-4488-4677-abf6-08bc931e195b"}, false},
		{"Base-Linear-AllHashes.aff4", []string{"2a497fe5-0221-4156-8b4d-176bebf7163f"}, false},
		{"Base-Allocated.aff4", []string{"e9cd53d3-b682-4f12-8045-86ba50a0239c"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, s, err := loadFile("images/" + tt.name)
			if err != nil {
				t.Fatal(err)
			}
			fsys, err := New(r, s)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			err = fstest.TestFS(fsys, tt.want[0])
			if err != nil {
				t.Fatal(err)
			}

			var names []string
			err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
				if path == "." {
					return nil
				}
				names = append(names, d.Name())
				/*
					dst, err := os.Create(strings.TrimPrefix(d.Name(), `aff4://`))
					if err != nil {
						return err
					}
					src, err := fsys.Open(path)
					if err != nil {
						return fmt.Errorf("x %w %s", err, path)
					}
					_, err = io.Copy(dst, src)
				*/
				return err
			})
			if err != nil {
				t.Error(err)
			}

			if !reflect.DeepEqual(names, tt.want) {
				t.Errorf("New() got = %v, want %v", names, tt.want)
			}

		})
	}
}
